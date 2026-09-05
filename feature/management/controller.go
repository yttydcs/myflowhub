package management

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/command"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

const contentTypeJSON = "application/json"

type Settings interface {
	Snapshot() protocol.ManagementConfigV1
	Replace(protocol.ManagementConfigUpdateV1) (protocol.ManagementConfigV1, error)
}

type Config struct {
	Node                *node.Node
	Admission           *auth.Admission
	EnrollmentAuthority *auth.EnrollmentAuthority
	AuthorityNodeID     protocol.NodeID
	Trust               *auth.TrustStore
	Policy              *auth.PolicyState
	Settings            Settings
	RevokeNode          func(protocol.NodeID) error
	Audit               *AuditLog
	Now                 func() time.Time
}

type Controller struct {
	mu              sync.Mutex
	node            *node.Node
	admission       *auth.Admission
	authority       *auth.EnrollmentAuthority
	authorityNodeID protocol.NodeID
	trust           *auth.TrustStore
	policy          *auth.PolicyState
	settings        Settings
	revoke          func(protocol.NodeID) error
	audit           *AuditLog
	now             func() time.Time
	startedAt       time.Time
	topology        *resource.Variable
	topologyQuery   *topologyResource
	health          *resource.Variable
	config          *resource.Variable
	auditFeed       *resource.Stream
	admissionStatus *resource.Variable
	lastError       string
}

func Register(config Config) (*Controller, error) {
	if config.Node == nil || config.Admission == nil || config.Trust == nil || config.Policy == nil || config.Settings == nil || config.RevokeNode == nil {
		return nil, errors.New("management node, stores, settings, and revoker are required")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Audit == nil {
		config.Audit = NewAuditLog(config.Now, 256)
	}
	value := &Controller{
		node: config.Node, admission: config.Admission, authority: config.EnrollmentAuthority, authorityNodeID: config.AuthorityNodeID, trust: config.Trust, policy: config.Policy,
		settings: config.Settings, revoke: config.RevokeNode, audit: config.Audit, now: config.Now, startedAt: config.Now().UTC(),
	}
	if value.authority != nil {
		if value.authorityNodeID != 0 && value.authorityNodeID != value.authority.NodeID() {
			return nil, errors.New("configured Admission Authority Node ID does not match the local Authority")
		}
		value.authorityNodeID = value.authority.NodeID()
	}
	resources, err := value.buildResources()
	if err != nil {
		return nil, err
	}
	registered := make([]protocol.ResourceID, 0, len(resources))
	for _, current := range resources {
		if err := config.Node.Registry().Register(current); err != nil {
			for index := len(registered) - 1; index >= 0; index-- {
				_ = config.Node.Registry().Remove(registered[index])
			}
			return nil, fmt.Errorf("register management resource %s: %w", current.Descriptor().ID.Name, err)
		}
		registered = append(registered, current.Descriptor().ID)
	}
	if err := value.audit.Bind(value.auditFeed); err != nil {
		for index := len(registered) - 1; index >= 0; index-- {
			_ = config.Node.Registry().Remove(registered[index])
		}
		return nil, fmt.Errorf("bind management audit stream: %w", err)
	}
	if err := value.Refresh(); err != nil && !errors.Is(err, protocol.ErrPayloadTooLarge) {
		for index := len(registered) - 1; index >= 0; index-- {
			_ = config.Node.Registry().Remove(registered[index])
		}
		return nil, err
	}
	return value, nil
}

func (c *Controller) Refresh() (err error) {
	if c == nil {
		return errors.New("management controller is required")
	}
	c.mu.Lock()
	publication, topologyErr := c.refreshTopologyLocked()
	defer func() {
		c.mu.Unlock()
		// Reentrant refreshes enqueue their newer snapshot; they never invoke
		// another observer round inside an older publication.
		if publication != nil {
			err = errors.Join(err, c.topologyQuery.publish(publication))
		}
	}()
	if topologyErr != nil {
		c.lastError = topologyErr.Error()
	} else {
		c.lastError = ""
	}
	if err := c.refreshConfigLocked(); err != nil {
		c.lastError = err.Error()
		return err
	}
	if err := c.refreshHealthLocked(); err != nil {
		c.lastError = err.Error()
		return err
	}
	if err := c.refreshAdmissionStatusLocked(); err != nil {
		c.lastError = err.Error()
		return err
	}
	if topologyErr == nil {
		c.lastError = ""
	}
	return topologyErr
}

func (c *Controller) buildResources() ([]resource.Resource, error) {
	owner := c.node.ID()
	topology, err := newJSONVariable(owner, protocol.BuiltinManagementTopology, protocol.SchemaManagementTopologyV1, "management.topology.read", protocol.ManagementTopologyV1{Version: 1, Epoch: 1, Nodes: []protocol.TopologyNodeV1{{NodeID: strconv.FormatUint(uint64(owner), 10), Role: "root", Generation: 1}}})
	if err != nil {
		return nil, err
	}
	health, err := newJSONVariable(owner, protocol.BuiltinManagementHealth, protocol.SchemaManagementHealthV1, "management.health.read", protocol.ManagementHealthV1{Version: 1, State: "starting", StartedAtUnixMS: c.startedAt.UnixMilli(), TopologyEpoch: 1})
	if err != nil {
		return nil, err
	}
	settings := c.settings.Snapshot()
	configVariable, err := newJSONVariable(owner, protocol.BuiltinManagementConfig, protocol.SchemaManagementConfigV1, "management.config.read", settings)
	if err != nil {
		return nil, err
	}
	auditFeed, err := resource.NewStream(resource.StreamDescriptor(protocol.ResourceID{Owner: owner, Name: protocol.BuiltinManagementAudit}, contentTypeJSON, protocol.SchemaManagementAuditV1, "management.audit.read", protocol.DefaultMaxPayload))
	if err != nil {
		return nil, err
	}
	c.topology, c.health, c.config, c.auditFeed = topology, health, configVariable, auditFeed
	c.topologyQuery, err = newTopologyResource(topology)
	if err != nil {
		return nil, err
	}
	issueInputSchema := protocol.SchemaManagementIssuePermitV1
	issueOutputSchema := protocol.SchemaProvisioningPermitV1
	if c.authority != nil {
		issueInputSchema = protocol.SchemaAdmissionIssuePermitV1
		issueOutputSchema = protocol.SchemaEnrollmentPermitV1
	}
	issue, err := c.command(protocol.BuiltinManagementIssuePermit, issueInputSchema, issueOutputSchema, "management.admission.issue", c.issuePermit)
	if err != nil {
		return nil, err
	}
	revokePermit, err := c.command(protocol.BuiltinManagementRevokePermit, protocol.SchemaManagementRevokePermitV1, protocol.SchemaManagementResultV1, "management.admission.revoke", c.revokePermit)
	if err != nil {
		return nil, err
	}
	revokeNode, err := c.command(protocol.BuiltinManagementRevokeNode, protocol.SchemaManagementRevokeV1, protocol.SchemaManagementResultV1, "management.node.revoke", c.revokeNode)
	if err != nil {
		return nil, err
	}
	updateConfig, err := c.command(protocol.BuiltinManagementConfigUpdate, protocol.SchemaManagementConfigUpdateV1, protocol.SchemaManagementConfigV1, "management.config.write", c.updateConfig)
	if err != nil {
		return nil, err
	}
	resources := []resource.Resource{c.topologyQuery, health, configVariable, auditFeed, issue, revokePermit, revokeNode, updateConfig}
	policyResources, err := c.buildPolicyResources()
	if err != nil {
		return nil, err
	}
	resources = append(resources, policyResources...)
	admissionResources, err := c.buildEnrollmentAuthorityResources()
	if err != nil {
		return nil, err
	}
	return append(resources, admissionResources...), nil
}

func (c *Controller) command(name, inputSchema, outputSchema, permission string, handler resource.CommandHandler) (*resource.Command, error) {
	return resource.NewCommand(resource.CommandDescriptorSchemas(protocol.ResourceID{Owner: c.node.ID(), Name: name}, contentTypeJSON, inputSchema, outputSchema, permission, protocol.DefaultMaxPayload), handler)
}

func (c *Controller) issuePermit(ctx context.Context, input []byte) ([]byte, error) {
	if c.authority != nil {
		var request protocol.AdmissionIssuePermitV1
		if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err == nil {
			if request.TTLMS > math.MaxInt64/int64(time.Millisecond) {
				return nil, errors.New("permit ttl_ms overflows duration")
			}
			targetID, _ := strconv.ParseUint(request.TargetNodeID, 10, 64)
			permit, err := c.authority.IssuePermit(request.RequestID, request.DevicePublicKeyFingerprint, protocol.NodeID(targetID), request.AllowDescendants, request.AdmissionProfile, time.Duration(request.TTLMS)*time.Millisecond)
			if err != nil {
				return nil, err
			}
			c.auditAdmissionOutcome(ctx, "management.admission.issue", protocol.BuiltinAdmissionIssuePermit, permit.PermitID, "active")
			if err := c.Refresh(); err != nil {
				return nil, err
			}
			return protocol.EncodeJSONPayload(&permit, protocol.DefaultMaxPayload)
		}
	}
	var request protocol.ManagementIssuePermitV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	if request.TTLMS > math.MaxInt64/int64(time.Millisecond) {
		return nil, errors.New("permit ttl_ms overflows duration")
	}
	childID, _ := strconv.ParseUint(request.ChildNodeID, 10, 64)
	key, _ := base64.RawStdEncoding.DecodeString(request.ChildPublicKey)
	permit, err := c.admission.Issue(protocol.NodeID(childID), ed25519.PublicKey(key), request.Role, time.Duration(request.TTLMS)*time.Millisecond)
	if err != nil {
		return nil, err
	}
	return protocol.EncodeJSONPayload(&permit, protocol.DefaultMaxPayload)
}

func (c *Controller) auditAdmissionOutcome(ctx context.Context, action auth.Action, resourceName, target, status string) {
	delegation, ok := command.DelegationFromContext(ctx)
	subject, subjectOK := delegation.Subject()
	if !ok || !subjectOK {
		return
	}
	c.audit.RecordOutcome(auth.Request{
		Subject: subject,
		Action:  action,
		Resource: protocol.ResourceID{
			Owner: c.node.ID(),
			Name:  resourceName,
		},
	}, target, status)
}

func (c *Controller) revokePermit(_ context.Context, input []byte) ([]byte, error) {
	var request protocol.ManagementRevokePermitV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	if err := c.admission.Revoke(request.PermitID); err != nil {
		return nil, err
	}
	return managementOK()
}

func (c *Controller) revokeNode(_ context.Context, input []byte) ([]byte, error) {
	var request protocol.ManagementRevokeV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	parsed, _ := strconv.ParseUint(request.NodeID, 10, 64)
	nodeID := protocol.NodeID(parsed)
	if err := c.revoke(nodeID); err != nil {
		return nil, err
	}
	_ = c.node.DisconnectPeer(nodeID)
	if err := c.Refresh(); err != nil {
		return nil, err
	}
	return managementOK()
}

func (c *Controller) updateConfig(_ context.Context, input []byte) ([]byte, error) {
	var request protocol.ManagementConfigUpdateV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	next, err := c.settings.Replace(request)
	if err != nil {
		return nil, err
	}
	if err := c.Refresh(); err != nil {
		return nil, err
	}
	return protocol.EncodeJSONPayload(&next, protocol.DefaultMaxPayload)
}

func decodePolicyRule(input []byte) (auth.Request, error) {
	var request protocol.ManagementPolicyRuleV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return auth.Request{}, err
	}
	subject, _ := strconv.ParseUint(request.Subject, 10, 64)
	owner, _ := strconv.ParseUint(request.ResourceNode, 10, 64)
	return auth.Request{
		Subject: protocol.NodeID(subject), Action: auth.Action(request.Action),
		Resource: protocol.ResourceID{Owner: protocol.NodeID(owner), Name: request.ResourceName},
	}, nil
}

func (c *Controller) refreshConfigLocked() error {
	payload, err := protocol.EncodeJSONPayload(configPointer(c.settings.Snapshot()), protocol.DefaultMaxPayload)
	if err != nil {
		return err
	}
	return setIfChanged(c.config, payload)
}

func (c *Controller) refreshHealthLocked() error {
	epoch := c.node.Tree().Epoch()
	if epoch == 0 {
		epoch = 1
	}
	stats := c.node.Stats()
	health := protocol.ManagementHealthV1{
		Version: 1, State: "running", StartedAtUnixMS: c.startedAt.UnixMilli(), TopologyEpoch: epoch,
		ActiveLinks: stats.ActiveLinks, ActiveSubscriptions: stats.ActiveSubscriptions, LastError: c.lastError,
		Components: map[string]string{
			"admission": fmt.Sprintf("generation:%d", c.admission.Generation()),
			"policy":    fmt.Sprintf("generation:%d", c.policy.Generation()),
			"trust":     fmt.Sprintf("generation:%d", c.trust.Generation()),
		},
	}
	payload, err := protocol.EncodeJSONPayload(&health, protocol.DefaultMaxPayload)
	if err != nil {
		return err
	}
	return setIfChanged(c.health, payload)
}

func newJSONVariable(owner protocol.NodeID, name, schema, permission string, initial protocol.ValidatedPayload) (*resource.Variable, error) {
	payload, err := protocol.EncodeJSONPayload(initial, protocol.DefaultMaxPayload)
	if err != nil {
		return nil, err
	}
	return resource.NewVariable(resource.VariableDescriptor(protocol.ResourceID{Owner: owner, Name: name}, contentTypeJSON, schema, permission, protocol.DefaultMaxPayload), payload)
}

func setIfChanged(variable *resource.Variable, payload []byte) error {
	if bytes.Equal(variable.Snapshot().Value, payload) {
		return nil
	}
	_, err := variable.Set(payload)
	return err
}

func configPointer(value protocol.ManagementConfigV1) *protocol.ManagementConfigV1 { return &value }

func managementOK() ([]byte, error) {
	return protocol.EncodeJSONPayload(&protocol.ManagementResultV1{Version: 1, Status: "ok"}, protocol.DefaultMaxPayload)
}
