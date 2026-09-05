package sdk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
)

func DecodeEvent(event Event, target protocol.ValidatedPayload) error {
	if target == nil {
		return errors.New("SDK event target is required")
	}
	if err := protocol.DecodeJSONPayload(event.Value, protocol.DefaultMaxPayload, target); err != nil {
		return &Error{Code: protocol.CodeMalformed, Message: fmt.Sprintf("decode %s event: %v", event.Resource.Name, err), cause: err}
	}
	return nil
}

// CollectionClient provides the standard Collection list/get operations for
// one Resource. Domain-specific get payloads should use OperatePayload with
// CapabilityGet instead of reinterpreting CollectionMemberV1.
type CollectionClient struct {
	client   *Client
	resource protocol.ResourceID
}

func (c *Client) Collection(resourceID protocol.ResourceID) (*CollectionClient, error) {
	if _, err := c.runtimeNode(); err != nil {
		return nil, err
	}
	if err := resourceID.Validate(); err != nil {
		return nil, fmt.Errorf("SDK collection resource: %w", err)
	}
	return &CollectionClient{client: c, resource: resourceID}, nil
}

func (c *CollectionClient) List(ctx context.Context, request protocol.CollectionListRequestV1) (protocol.CollectionPageV1, error) {
	var response protocol.CollectionPageV1
	if c == nil || c.client == nil {
		return response, errors.New("SDK collection client is required")
	}
	err := c.client.OperatePayload(ctx, c.resource, protocol.CapabilityList, &request, &response)
	return response, err
}

func (c *CollectionClient) Get(ctx context.Context, request protocol.CollectionMemberRequestV1) (protocol.CollectionMemberV1, error) {
	var response protocol.CollectionMemberV1
	if c == nil || c.client == nil {
		return response, errors.New("SDK collection client is required")
	}
	err := c.client.OperatePayload(ctx, c.resource, protocol.CapabilityGet, &request, &response)
	return response, err
}

type TopicClient struct {
	client   *Client
	resource protocol.ResourceID
	schema   string
}

func (c *Client) Topic(resourceID protocol.ResourceID, schema string) (*TopicClient, error) {
	if _, err := c.runtimeNode(); err != nil {
		return nil, err
	}
	if err := resourceID.Validate(); err != nil {
		return nil, err
	}
	if schema == "" {
		return nil, errors.New("topic schema is required")
	}
	return &TopicClient{client: c, resource: resourceID, schema: schema}, nil
}

func (c *TopicClient) Publish(ctx context.Context, payload []byte) error {
	_, err := c.client.Operate(ctx, c.resource, protocol.CapabilityPublish, c.schema, payload)
	return err
}

func (c *TopicClient) Events(ctx context.Context, lease time.Duration, queue int) (*Subscription, error) {
	return c.client.SubscribeCapability(ctx, c.resource, protocol.CapabilitySubscribe, lease, queue)
}

type ManagementClient struct {
	client *Client
	owner  protocol.NodeID
}

func (c *Client) Management(owner protocol.NodeID) (*ManagementClient, error) {
	if _, err := c.runtimeNode(); err != nil {
		return nil, err
	}
	if err := owner.Validate(); err != nil {
		return nil, err
	}
	return &ManagementClient{client: c, owner: owner}, nil
}

func (c *ManagementClient) Topology(ctx context.Context) (protocol.ManagementTopologyV1, error) {
	var value protocol.ManagementTopologyV1
	err := c.client.DecodeSnapshot(ctx, c.id(protocol.BuiltinManagementTopology), &value)
	return value, err
}

func (c *ManagementClient) Health(ctx context.Context) (protocol.ManagementHealthV1, error) {
	var value protocol.ManagementHealthV1
	err := c.client.DecodeSnapshot(ctx, c.id(protocol.BuiltinManagementHealth), &value)
	return value, err
}

func (c *ManagementClient) Config(ctx context.Context) (protocol.ManagementConfigV1, error) {
	var value protocol.ManagementConfigV1
	err := c.client.DecodeSnapshot(ctx, c.id(protocol.BuiltinManagementConfig), &value)
	return value, err
}

func (c *ManagementClient) IssuePermit(ctx context.Context, request protocol.ManagementIssuePermitV1) (protocol.ProvisioningPermitV1, error) {
	var response protocol.ProvisioningPermitV1
	err := c.client.InvokePayload(ctx, c.id(protocol.BuiltinManagementIssuePermit), &request, &response)
	return response, err
}

func (c *ManagementClient) RevokePermit(ctx context.Context, request protocol.ManagementRevokePermitV1) error {
	var response protocol.ManagementResultV1
	return c.client.InvokePayload(ctx, c.id(protocol.BuiltinManagementRevokePermit), &request, &response)
}

func (c *ManagementClient) RevokeNode(ctx context.Context, request protocol.ManagementRevokeV1) error {
	var response protocol.ManagementResultV1
	return c.client.InvokePayload(ctx, c.id(protocol.BuiltinManagementRevokeNode), &request, &response)
}

func (c *ManagementClient) GrantPolicy(ctx context.Context, request protocol.ManagementPolicyRuleV1) error {
	var response protocol.ManagementResultV1
	return c.client.InvokePayload(ctx, c.id(protocol.BuiltinManagementPolicyGrant), &request, &response)
}

func (c *ManagementClient) RevokePolicy(ctx context.Context, request protocol.ManagementPolicyRuleV1) error {
	var response protocol.ManagementResultV1
	return c.client.InvokePayload(ctx, c.id(protocol.BuiltinManagementPolicyRevoke), &request, &response)
}

func (c *ManagementClient) UpdateConfig(ctx context.Context, request protocol.ManagementConfigUpdateV1) (protocol.ManagementConfigV1, error) {
	var response protocol.ManagementConfigV1
	err := c.client.InvokePayload(ctx, c.id(protocol.BuiltinManagementConfigUpdate), &request, &response)
	return response, err
}

func (c *ManagementClient) Audit(ctx context.Context, lease time.Duration, queue int) (*Subscription, error) {
	return c.client.Subscribe(ctx, c.id(protocol.BuiltinManagementAudit), lease, queue)
}

func (c *ManagementClient) id(name string) protocol.ResourceID {
	return protocol.ResourceID{Owner: c.owner, Name: name}
}

// PolicyClient is a typed facade over the Authority-owned policy Collections.
// It uses the same attached Client and Node routing path as generic Operate.
type PolicyClient struct {
	client      *Client
	owner       protocol.NodeID
	definitions *CollectionClient
	bindings    *CollectionClient
	grants      *CollectionClient
}

func (c *Client) Policies(owner protocol.NodeID) (*PolicyClient, error) {
	if _, err := c.runtimeNode(); err != nil {
		return nil, err
	}
	if err := owner.Validate(); err != nil {
		return nil, fmt.Errorf("SDK policy owner: %w", err)
	}
	definitions, err := c.Collection(protocol.ResourceID{Owner: owner, Name: protocol.BuiltinPolicyDefinitions})
	if err != nil {
		return nil, err
	}
	bindings, err := c.Collection(protocol.ResourceID{Owner: owner, Name: protocol.BuiltinPolicyBindings})
	if err != nil {
		return nil, err
	}
	grants, err := c.Collection(protocol.ResourceID{Owner: owner, Name: protocol.BuiltinPolicyGrants})
	if err != nil {
		return nil, err
	}
	return &PolicyClient{client: c, owner: owner, definitions: definitions, bindings: bindings, grants: grants}, nil
}

func (c *PolicyClient) ListDefinitions(ctx context.Context, request protocol.CollectionListRequestV1) (protocol.CollectionPageV1, error) {
	if err := c.validate(); err != nil {
		return protocol.CollectionPageV1{}, err
	}
	return c.definitions.List(ctx, request)
}

func (c *PolicyClient) ListBindings(ctx context.Context, request protocol.CollectionListRequestV1) (protocol.CollectionPageV1, error) {
	if err := c.validate(); err != nil {
		return protocol.CollectionPageV1{}, err
	}
	return c.bindings.List(ctx, request)
}

func (c *PolicyClient) ListGrants(ctx context.Context, request protocol.CollectionListRequestV1) (protocol.CollectionPageV1, error) {
	if err := c.validate(); err != nil {
		return protocol.CollectionPageV1{}, err
	}
	return c.grants.List(ctx, request)
}

func (c *PolicyClient) GetDefinition(ctx context.Context, request protocol.CollectionMemberRequestV1) (protocol.PolicyDefinitionV1, error) {
	var response protocol.PolicyDefinitionV1
	err := c.operate(ctx, protocol.BuiltinPolicyDefinitions, protocol.CapabilityGet, &request, &response)
	return response, err
}

func (c *PolicyClient) CreateDefinition(ctx context.Context, request protocol.PolicyDefinitionPutV1) (protocol.PolicyDefinitionV1, error) {
	var response protocol.PolicyDefinitionV1
	err := c.operate(ctx, protocol.BuiltinPolicyDefinitions, protocol.CapabilityCreate, &request, &response)
	return response, err
}

func (c *PolicyClient) UpdateDefinition(ctx context.Context, request protocol.PolicyDefinitionPutV1) (protocol.PolicyDefinitionV1, error) {
	var response protocol.PolicyDefinitionV1
	err := c.operate(ctx, protocol.BuiltinPolicyDefinitions, protocol.CapabilityUpdate, &request, &response)
	return response, err
}

func (c *PolicyClient) DeleteDefinition(ctx context.Context, request protocol.PolicyDefinitionDeleteV1) error {
	var response protocol.ManagementResultV1
	return c.operate(ctx, protocol.BuiltinPolicyDefinitions, protocol.CapabilityDelete, &request, &response)
}

func (c *PolicyClient) GetBinding(ctx context.Context, request protocol.CollectionMemberRequestV1) (protocol.PolicyBindingV1, error) {
	var response protocol.PolicyBindingV1
	err := c.operate(ctx, protocol.BuiltinPolicyBindings, protocol.CapabilityGet, &request, &response)
	return response, err
}

func (c *PolicyClient) CreateBinding(ctx context.Context, request protocol.PolicyBindingCreateV1) (protocol.PolicyBindingV1, error) {
	var response protocol.PolicyBindingV1
	err := c.operate(ctx, protocol.BuiltinPolicyBindings, protocol.CapabilityCreate, &request, &response)
	return response, err
}

func (c *PolicyClient) RevokeBinding(ctx context.Context, request protocol.PolicyBindingRevokeV1) error {
	var response protocol.ManagementResultV1
	return c.operate(ctx, protocol.BuiltinPolicyBindings, protocol.CapabilityRevoke, &request, &response)
}

func (c *PolicyClient) Evaluate(ctx context.Context, request protocol.PolicyEvaluateRequestV1) (protocol.PolicyEvaluationV1, error) {
	var response protocol.PolicyEvaluationV1
	err := c.operate(ctx, protocol.BuiltinPolicyBindings, protocol.CapabilityEvaluate, &request, &response)
	return response, err
}

func (c *PolicyClient) GetGrant(ctx context.Context, request protocol.CollectionMemberRequestV1) (protocol.PolicyGrantV1, error) {
	var response protocol.PolicyGrantV1
	err := c.operate(ctx, protocol.BuiltinPolicyGrants, protocol.CapabilityGet, &request, &response)
	return response, err
}

func (c *PolicyClient) CreateGrant(ctx context.Context, request protocol.PolicyGrantV1) (protocol.PolicyGrantV1, error) {
	var response protocol.PolicyGrantV1
	err := c.operate(ctx, protocol.BuiltinPolicyGrants, protocol.CapabilityCreate, &request, &response)
	return response, err
}

func (c *PolicyClient) RevokeGrant(ctx context.Context, request protocol.PolicyGrantV1) error {
	var response protocol.ManagementResultV1
	return c.operate(ctx, protocol.BuiltinPolicyGrants, protocol.CapabilityRevoke, &request, &response)
}

func (c *PolicyClient) operate(ctx context.Context, name string, capability protocol.CapabilityID, request, response protocol.ValidatedPayload) error {
	if err := c.validate(); err != nil {
		return err
	}
	return c.client.OperatePayload(ctx, protocol.ResourceID{Owner: c.owner, Name: name}, capability, request, response)
}

func (c *PolicyClient) validate() error {
	if c == nil || c.client == nil || c.definitions == nil || c.bindings == nil || c.grants == nil {
		return errors.New("SDK policy client is required")
	}
	return nil
}

type NotificationClient struct {
	client *Client
	owner  protocol.NodeID
}

func (c *Client) Notifications(owner protocol.NodeID) (*NotificationClient, error) {
	if _, err := c.runtimeNode(); err != nil {
		return nil, err
	}
	if err := owner.Validate(); err != nil {
		return nil, err
	}
	return &NotificationClient{client: c, owner: owner}, nil
}

func (c *NotificationClient) Publish(ctx context.Context, request protocol.NotificationPublishV1) (protocol.NotificationEventV1, error) {
	var response protocol.NotificationEventV1
	err := c.client.InvokePayload(ctx, protocol.ResourceID{Owner: c.owner, Name: protocol.BuiltinNotificationPublish}, &request, &response)
	return response, err
}

func (c *NotificationClient) Events(ctx context.Context, lease time.Duration, queue int) (*Subscription, error) {
	return c.client.Subscribe(ctx, protocol.ResourceID{Owner: c.owner, Name: protocol.BuiltinNotificationEvents}, lease, queue)
}

type FileClient struct {
	client *Client
	owner  protocol.NodeID
}

func (c *Client) Files(owner protocol.NodeID) (*FileClient, error) {
	if _, err := c.runtimeNode(); err != nil {
		return nil, err
	}
	if err := owner.Validate(); err != nil {
		return nil, err
	}
	return &FileClient{client: c, owner: owner}, nil
}

func (c *FileClient) Transfers(ctx context.Context) (protocol.FileTransfersV1, error) {
	var value protocol.FileTransfersV1
	err := c.client.DecodeSnapshot(ctx, c.id(protocol.BuiltinFileTransfers), &value)
	return value, err
}

func (c *FileClient) Progress(ctx context.Context, lease time.Duration, queue int) (*Subscription, error) {
	return c.client.Subscribe(ctx, c.id(protocol.BuiltinFileProgress), lease, queue)
}

func (c *FileClient) UploadFile(ctx context.Context, sourcePath, destination, contentType string, chunkSize int, lifetime time.Duration) (result protocol.FileProgressV1, err error) {
	if ctx == nil {
		return result, errors.New("SDK file upload context is required")
	}
	if chunkSize <= 0 {
		chunkSize = protocol.MaxFileChunkBytes
	}
	if chunkSize > protocol.MaxFileChunkBytes || lifetime <= 0 {
		return result, errors.New("SDK file upload chunk size or lifetime is invalid")
	}
	file, err := os.Open(sourcePath)
	if err != nil {
		return result, fmt.Errorf("open upload source: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return result, errors.New("upload source must be a regular file")
	}
	hash := sha256.New()
	if _, err := copyContext(ctx, hash, file); err != nil {
		return result, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return result, err
	}
	id, err := protocol.NewMessageID()
	if err != nil {
		return result, err
	}
	offer := protocol.FileOfferV1{
		Version: 1, TransferID: id.String(), Path: destination, Size: info.Size(), SHA256: hex.EncodeToString(hash.Sum(nil)),
		ChunkSize: chunkSize, ContentType: contentType, ExpiresAtUnixMS: time.Now().Add(lifetime).UnixMilli(),
	}
	offerPayload, err := protocol.EncodeJSONPayload(&offer, protocol.DefaultMaxPayload)
	if err != nil {
		return result, err
	}
	runtime, err := c.client.runtimeNode()
	if err != nil {
		return result, err
	}
	session, err := runtime.OpenSession(ctx, c.id(protocol.BuiltinFileUpload), protocol.CapabilityOpen, protocol.SchemaFileOfferV1, offerPayload)
	if err != nil {
		return result, err
	}
	completed := false
	defer func() {
		if !completed {
			cancelCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, _ = session.Close(cancelCtx, false, "text/plain", []byte("upload aborted"))
		}
	}()
	buffer := make([]byte, chunkSize)
	var offset int64
	for {
		count, readErr := file.Read(buffer)
		if count > 0 {
			data := append([]byte(nil), buffer[:count]...)
			digest := sha256.Sum256(data)
			operation, sendErr := session.Send(ctx, offset, data, hex.EncodeToString(digest[:]))
			if sendErr == nil {
				sendErr = protocol.DecodeJSONPayload(operation.Payload, protocol.DefaultMaxPayload, &result)
			}
			err = sendErr
			if err != nil {
				return result, err
			}
			offset += int64(count)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return result, readErr
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}
	}
	operation, err := session.Close(ctx, true, "", nil)
	if err == nil {
		err = protocol.DecodeJSONPayload(operation.Payload, protocol.DefaultMaxPayload, &result)
	}
	if err == nil {
		completed = true
	}
	return result, err
}

func (c *FileClient) id(name string) protocol.ResourceID {
	return protocol.ResourceID{Owner: c.owner, Name: name}
}

func copyContext(ctx context.Context, destination io.Writer, source io.Reader) (int64, error) {
	buffer := make([]byte, 64<<10)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		count, readErr := source.Read(buffer)
		if count > 0 {
			written, err := destination.Write(buffer[:count])
			total += int64(written)
			if err != nil {
				return total, err
			}
			if written != count {
				return total, io.ErrShortWrite
			}
		}
		if errors.Is(readErr, io.EOF) {
			return total, nil
		}
		if readErr != nil {
			return total, readErr
		}
	}
}
