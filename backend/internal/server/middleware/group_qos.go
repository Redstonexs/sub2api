package middleware

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// bindGroupQoSDecision resolves the group's QoS degradation tier for this
// requester and binds it to the request context.
//
// It runs once per request, here rather than in the handlers, because the
// decision has four separate downstream consumers (model reroute, reasoning
// effort ceiling, RPM squeeze, hard block) and resolving it more than once would
// mean several counter reads per request — and could let two consumers disagree
// if the counter moved in between.
//
// Both API-key auth middlewares must call this: Gemini routes authenticate
// through APIKeyAuthWithSubscriptionGoogle, so binding in only one place would
// silently leave a whole platform unprotected.
func bindGroupQoSDecision(c *gin.Context, qosService *service.GroupQoSService, apiKey *service.APIKey) {
	if qosService == nil || apiKey == nil || apiKey.User == nil {
		return
	}
	if !service.GroupQoSEligible(apiKey.Group) {
		return
	}
	decision := qosService.ResolveDecision(c.Request.Context(), apiKey.User.ID, apiKey.Group)
	if decision == nil {
		return
	}
	c.Request = c.Request.WithContext(service.WithGroupQoSDecision(c.Request.Context(), decision))
	c.Set(string(ContextKeyGroupQoSDecision), decision)

	// Degradation is otherwise invisible: the client just sees quieter, cheaper
	// output and has no way to tell it apart from a model regression. These
	// headers make it diagnosable without exposing the ladder's thresholds.
	c.Header("X-Sub2API-QoS-Tier", strconv.Itoa(decision.TierIndex+1))
	c.Header("X-Sub2API-QoS-Window", decision.Window)
}

// GetGroupQoSDecisionFromContext returns the tier in effect for this request,
// or nil when the request is not degraded.
func GetGroupQoSDecisionFromContext(c *gin.Context) *service.GroupQoSDecision {
	if c == nil {
		return nil
	}
	value, exists := c.Get(string(ContextKeyGroupQoSDecision))
	if !exists {
		return nil
	}
	decision, _ := value.(*service.GroupQoSDecision)
	return decision
}
