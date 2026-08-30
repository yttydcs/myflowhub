package management

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

type AuditLog struct {
	mu      sync.Mutex
	now     func() time.Time
	limit   int
	pending [][]byte
	stream  *resource.Stream
}

func NewAuditLog(now func() time.Time, pendingLimit int) *AuditLog {
	if now == nil {
		now = time.Now
	}
	if pendingLimit <= 0 {
		pendingLimit = 256
	}
	return &AuditLog{now: now, limit: pendingLimit}
}

func (a *AuditLog) Bind(stream *resource.Stream) error {
	if a == nil || stream == nil {
		return errors.New("management audit log and stream are required")
	}
	a.mu.Lock()
	if a.stream != nil {
		a.mu.Unlock()
		return errors.New("management audit stream is already bound")
	}
	a.stream = stream
	pending := a.pending
	a.pending = nil
	for _, payload := range pending {
		if _, err := stream.Publish(payload); err != nil {
			a.mu.Unlock()
			return err
		}
	}
	a.mu.Unlock()
	return nil
}

func (a *AuditLog) RecordDecision(request auth.Request, decisionErr error) {
	if a == nil {
		return
	}
	decision := "allow"
	detail := ""
	if decisionErr != nil {
		decision = "deny"
		detail = decisionErr.Error()
		for len(detail) > 2048 {
			_, size := utf8.DecodeLastRuneInString(detail)
			detail = detail[:len(detail)-size]
		}
	}
	event := protocol.ManagementAuditV1{
		Version: 1, TimeUnixMS: a.now().UTC().UnixMilli(), Subject: strconv.FormatUint(uint64(request.Subject), 10),
		Action: string(request.Action), ResourceNode: strconv.FormatUint(uint64(request.Resource.Owner), 10),
		ResourceName: request.Resource.Name, Decision: decision, Detail: detail,
	}
	a.record(event)
}

func (a *AuditLog) RecordOutcome(request auth.Request, target, status string) {
	if a == nil {
		return
	}
	event := protocol.ManagementAuditV1{
		Version: 1, TimeUnixMS: a.now().UTC().UnixMilli(), Subject: strconv.FormatUint(uint64(request.Subject), 10),
		Action: string(request.Action), ResourceNode: strconv.FormatUint(uint64(request.Resource.Owner), 10),
		ResourceName: request.Resource.Name, Decision: "allow", Target: target, Status: status,
	}
	a.record(event)
}

func (a *AuditLog) record(event protocol.ManagementAuditV1) {
	if a == nil {
		return
	}
	payload, err := protocol.EncodeJSONPayload(&event, protocol.DefaultMaxPayload)
	if err != nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stream != nil {
		_, _ = a.stream.Publish(payload)
		return
	}
	if len(a.pending) == a.limit {
		copy(a.pending, a.pending[1:])
		a.pending[len(a.pending)-1] = payload
		return
	}
	a.pending = append(a.pending, payload)
}

type auditedPolicy struct {
	base  auth.Policy
	audit *AuditLog
}

func AuditPolicy(base auth.Policy, audit *AuditLog) auth.Policy {
	if base == nil {
		base = auth.NewStaticPolicy()
	}
	return &auditedPolicy{base: base, audit: audit}
}

func (p *auditedPolicy) Authorize(ctx context.Context, request auth.Request) error {
	err := p.base.Authorize(ctx, request)
	p.audit.RecordDecision(request, err)
	return err
}

func (p *auditedPolicy) Generation() uint64 { return auth.PolicyGeneration(p.base) }

func (p *auditedPolicy) WatchGeneration(observer func(uint64)) (uint64, func(), error) {
	if generated, ok := p.base.(interface {
		WatchGeneration(func(uint64)) (uint64, func(), error)
	}); ok {
		return generated.WatchGeneration(observer)
	}
	if observer == nil {
		return 0, nil, errors.New("policy generation observer is required")
	}
	return p.Generation(), func() {}, nil
}
