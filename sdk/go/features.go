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

func (c *FileClient) Offer(ctx context.Context, request protocol.FileOfferV1) (protocol.FileProgressV1, error) {
	var response protocol.FileProgressV1
	err := c.client.InvokePayload(ctx, c.id(protocol.BuiltinFileOffer), &request, &response)
	return response, err
}

func (c *FileClient) Chunk(ctx context.Context, request protocol.FileChunkV1) (protocol.FileProgressV1, error) {
	var response protocol.FileProgressV1
	err := c.client.InvokePayload(ctx, c.id(protocol.BuiltinFileChunk), &request, &response)
	return response, err
}

func (c *FileClient) Complete(ctx context.Context, request protocol.FileCompleteV1) (protocol.FileProgressV1, error) {
	var response protocol.FileProgressV1
	err := c.client.InvokePayload(ctx, c.id(protocol.BuiltinFileComplete), &request, &response)
	return response, err
}

func (c *FileClient) Cancel(ctx context.Context, request protocol.FileCancelV1) (protocol.FileProgressV1, error) {
	var response protocol.FileProgressV1
	err := c.client.InvokePayload(ctx, c.id(protocol.BuiltinFileCancel), &request, &response)
	return response, err
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
	if result, err = c.Offer(ctx, offer); err != nil {
		return result, err
	}
	completed := false
	defer func() {
		if !completed {
			cancelCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, _ = c.Cancel(cancelCtx, protocol.FileCancelV1{Version: 1, TransferID: offer.TransferID, Reason: "upload aborted"})
		}
	}()
	buffer := make([]byte, chunkSize)
	var offset int64
	for {
		count, readErr := file.Read(buffer)
		if count > 0 {
			data := append([]byte(nil), buffer[:count]...)
			digest := sha256.Sum256(data)
			result, err = c.Chunk(ctx, protocol.FileChunkV1{Version: 1, TransferID: offer.TransferID, Offset: offset, Data: data, SHA256: hex.EncodeToString(digest[:])})
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
	result, err = c.Complete(ctx, protocol.FileCompleteV1{Version: 1, TransferID: offer.TransferID, Size: offer.Size, SHA256: offer.SHA256})
	if err == nil {
		completed = true
	}
	return result, err
}

func (c *FileClient) id(name string) protocol.ResourceID {
	return protocol.ResourceID{Owner: c.owner, Name: name}
}

type FlowClient struct {
	client *Client
	owner  protocol.NodeID
}

func (c *Client) Flows(owner protocol.NodeID) (*FlowClient, error) {
	if _, err := c.runtimeNode(); err != nil {
		return nil, err
	}
	if err := owner.Validate(); err != nil {
		return nil, err
	}
	return &FlowClient{client: c, owner: owner}, nil
}

func (c *FlowClient) Definitions(ctx context.Context) (protocol.FlowDefinitionsV1, error) {
	var value protocol.FlowDefinitionsV1
	err := c.client.DecodeSnapshot(ctx, c.id(protocol.BuiltinFlowDefinitions), &value)
	return value, err
}

func (c *FlowClient) Runs(ctx context.Context) (protocol.FlowRunsV1, error) {
	var value protocol.FlowRunsV1
	err := c.client.DecodeSnapshot(ctx, c.id(protocol.BuiltinFlowRuns), &value)
	return value, err
}

func (c *FlowClient) Events(ctx context.Context, lease time.Duration, queue int) (*Subscription, error) {
	return c.client.Subscribe(ctx, c.id(protocol.BuiltinFlowEvents), lease, queue)
}

func (c *FlowClient) Create(ctx context.Context, request protocol.FlowDefinitionV1) (protocol.FlowDefinitionV1, error) {
	var response protocol.FlowDefinitionV1
	err := c.client.InvokePayload(ctx, c.id(protocol.BuiltinFlowCreate), &request, &response)
	return response, err
}

func (c *FlowClient) Update(ctx context.Context, request protocol.FlowDefinitionV1) (protocol.FlowDefinitionV1, error) {
	var response protocol.FlowDefinitionV1
	err := c.client.InvokePayload(ctx, c.id(protocol.BuiltinFlowUpdate), &request, &response)
	return response, err
}

func (c *FlowClient) Run(ctx context.Context, request protocol.FlowRunV1) (protocol.FlowRunSummaryV1, error) {
	var response protocol.FlowRunSummaryV1
	err := c.client.InvokePayload(ctx, c.id(protocol.BuiltinFlowRun), &request, &response)
	return response, err
}

func (c *FlowClient) Cancel(ctx context.Context, request protocol.FlowCancelV1) (protocol.FlowRunSummaryV1, error) {
	var response protocol.FlowRunSummaryV1
	err := c.client.InvokePayload(ctx, c.id(protocol.BuiltinFlowCancel), &request, &response)
	return response, err
}

func (c *FlowClient) Archive(ctx context.Context, request protocol.FlowArchiveV1) error {
	var response protocol.FlowArchiveV1
	return c.client.InvokePayload(ctx, c.id(protocol.BuiltinFlowArchive), &request, &response)
}

func (c *FlowClient) id(name string) protocol.ResourceID {
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
