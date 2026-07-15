package message

import (
	"context"
	"testing"

	"ziziphus/pkg/model"
)

func TestContains(t *testing.T) {
	t.Run("finds item in slice", func(t *testing.T) {
		if !contains([]string{"a", "b", "c"}, "b") {
			t.Error("expected true for 'b' in [a b c]")
		}
	})

	t.Run("returns false for missing item", func(t *testing.T) {
		if contains([]string{"a", "b", "c"}, "z") {
			t.Error("expected false for 'z' in [a b c]")
		}
	})

	t.Run("empty slice returns false", func(t *testing.T) {
		if contains([]string{}, "x") {
			t.Error("expected false for empty slice")
		}
	})

	t.Run("nil slice returns false", func(t *testing.T) {
		if contains(nil, "x") {
			t.Error("expected false for nil slice")
		}
	})

	t.Run("empty string matches empty string", func(t *testing.T) {
		if !contains([]string{""}, "") {
			t.Error("expected true for empty string in slice containing empty string")
		}
	})
}

func TestRoute_AgentMentionModeSkipsNonMentioned(t *testing.T) {
	router, convMgr, sessGtr := defaultRouterGroupFixture()

	// Add an agent member with WakeModeMention.
	convMgr.members["group_1"] = append(convMgr.members["group_1"], &model.ConvMember{
		ConvID:   "group_1",
		UserID:   "agent_1",
		UserType: model.UserAgent,
		WakeMode: model.WakeModeMention,
	})
	sessGtr.sessions["agent_1"] = []string{"sess_a1"}

	msg := &model.Message{
		MsgID:    1001,
		ConvID:   "group_1",
		SenderID: "user_a",
		Body:     "hello",
		Mention:  []string{}, // agent not mentioned
	}
	targets := router.Route(context.Background(), msg)

	// Should NOT include agent_1 since it wasn't mentioned
	for _, tr := range targets {
		if tr.UserID == "agent_1" {
			t.Error("agent_1 should not be a target when not mentioned and WakeModeMention")
		}
	}
}

func TestRoute_AgentMentionModeIncludesMentioned(t *testing.T) {
	router, convMgr, sessGtr := defaultRouterGroupFixture()

	convMgr.members["group_1"] = append(convMgr.members["group_1"], &model.ConvMember{
		ConvID:   "group_1",
		UserID:   "agent_1",
		UserType: model.UserAgent,
		WakeMode: model.WakeModeMention,
	})
	sessGtr.sessions["agent_1"] = []string{"sess_a1"}

	msg := &model.Message{
		MsgID:    1001,
		ConvID:   "group_1",
		SenderID: "user_a",
		Body:     "hello @agent",
		Mention:  []string{"agent_1"}, // agent IS mentioned
	}
	targets := router.Route(context.Background(), msg)

	found := false
	for _, tr := range targets {
		if tr.UserID == "agent_1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("agent_1 should be a target when mentioned")
	}
}

func TestRoute_MemberNoSessions(t *testing.T) {
	router, convMgr, _ := defaultRouterGroupFixture()

	// user_a has no sessions configured
	sessGtr := &mockSessionGetter{} // empty sessions

	// Override the router with a new session getter
	router = NewRouter(sessGtr, convMgr, &mockConnRegistry{})

	msg := &model.Message{
		MsgID:    1001,
		ConvID:   "group_1",
		SenderID: "user_b",
		Body:     "hello",
	}
	targets := router.Route(context.Background(), msg)
	// user_b and user_c also have no sessions now, so no targets
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestRouteP2P_GetMembersError(t *testing.T) {
	router, convMgr, _ := defaultRouterFixture()

	// Remove members to trigger error in routeP2P
	delete(convMgr.members, "user_a:user_b")

	msg := &model.Message{
		MsgID:    1001,
		ConvID:   "user_a:user_b",
		SenderID: "user_a",
	}
	targets := router.Route(context.Background(), msg)
	if targets != nil {
		t.Errorf("expected nil targets on GetMembers error, got %d", len(targets))
	}
}

func TestRouteGroup_GetMembersError(t *testing.T) {
	router, convMgr, _ := defaultRouterGroupFixture()

	delete(convMgr.members, "group_1")

	msg := &model.Message{
		MsgID:    1001,
		ConvID:   "group_1",
		SenderID: "user_a",
	}
	targets := router.Route(context.Background(), msg)
	if targets != nil {
		t.Errorf("expected nil targets on GetMembers error, got %d", len(targets))
	}
}
