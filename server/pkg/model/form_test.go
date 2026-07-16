package model

import (
	"encoding/json"
	"testing"
)

func TestContentType_FormConstants(t *testing.T) {
	if ContentForm != 10 {
		t.Errorf("ContentForm = %d; want 10", ContentForm)
	}
	if ContentFormResponse != 11 {
		t.Errorf("ContentFormResponse = %d; want 11", ContentFormResponse)
	}
}

func TestFormDefinitionBody_RoundTrip(t *testing.T) {
	original := FormDefinitionBody{
		FormID:         "uuid-1234",
		Type:           FormTypeContactRequest,
		Title:          "好友申请",
		Description:    "申请加你为好友",
		FromUserID:     "user_a",
		FromUserName:   "张三",
		FromUserAvatar: "/avatar/a.jpg",
		RequestID:      1,
		Message:        "我是xxx",
		Actions: []FormAction{
			{Action: "approve", Label: "通过", Style: FormActionStylePrimary},
			{Action: "reject", Label: "拒绝", Style: FormActionStyleDanger},
		},
		Status:    FormStatusActive,
		CreatedAt: 1719532800000,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal FormDefinitionBody: %v", err)
	}

	var decoded FormDefinitionBody
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal FormDefinitionBody: %v", err)
	}

	if decoded.FormID != original.FormID {
		t.Errorf("FormID = %q; want %q", decoded.FormID, original.FormID)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type = %q; want %q", decoded.Type, original.Type)
	}
	if decoded.Title != original.Title {
		t.Errorf("Title = %q; want %q", decoded.Title, original.Title)
	}
	if decoded.FromUserID != original.FromUserID {
		t.Errorf("FromUserID = %q; want %q", decoded.FromUserID, original.FromUserID)
	}
	if decoded.RequestID != original.RequestID {
		t.Errorf("RequestID = %d; want %d", decoded.RequestID, original.RequestID)
	}
	if len(decoded.Actions) != 2 {
		t.Fatalf("len(Actions) = %d; want 2", len(decoded.Actions))
	}
	if decoded.Actions[0].Action != "approve" {
		t.Errorf("Actions[0].Action = %q; want approve", decoded.Actions[0].Action)
	}
	if decoded.Status != FormStatusActive {
		t.Errorf("Status = %q; want %q", decoded.Status, FormStatusActive)
	}
}

func TestFormResponseBody_RoundTrip(t *testing.T) {
	original := FormResponseBody{
		FormMsgID:     123456789,
		RequestID:     1,
		Action:        "approve",
		ResponderID:   "user_b",
		ResponderName: "李四",
		SubmittedAt:   1719532900000,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal FormResponseBody: %v", err)
	}

	var decoded FormResponseBody
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal FormResponseBody: %v", err)
	}

	if decoded.FormMsgID != original.FormMsgID {
		t.Errorf("FormMsgID = %d; want %d", decoded.FormMsgID, original.FormMsgID)
	}
	if decoded.RequestID != original.RequestID {
		t.Errorf("RequestID = %d; want %d", decoded.RequestID, original.RequestID)
	}
	if decoded.Action != "approve" {
		t.Errorf("Action = %q; want approve", decoded.Action)
	}
	if decoded.ResponderID != original.ResponderID {
		t.Errorf("ResponderID = %q; want %q", decoded.ResponderID, original.ResponderID)
	}
}

func TestContactRequestStatus_Constants(t *testing.T) {
	if ContactRequestPending != 0 {
		t.Errorf("ContactRequestPending = %d; want 0", ContactRequestPending)
	}
	if ContactRequestApproved != 1 {
		t.Errorf("ContactRequestApproved = %d; want 1", ContactRequestApproved)
	}
	if ContactRequestRejected != 2 {
		t.Errorf("ContactRequestRejected = %d; want 2", ContactRequestRejected)
	}
}

func TestConvType_System(t *testing.T) {
	if ConvSystem != 3 {
		t.Errorf("ConvSystem = %d; want 3", ConvSystem)
	}
}

func TestFormDefinitionBody_WithFields(t *testing.T) {
	original := FormDefinitionBody{
		FormID: "uuid-fields",
		Type:   "survey",
		Title:  "问卷",
		Fields: []FormField{
			{
				FieldID:  "f1",
				Type:     FormFieldRadio,
				Label:    "选择",
				Required: true,
				Options:  []string{"A", "B", "C"},
			},
			{
				FieldID:     "f2",
				Type:        FormFieldTextarea,
				Label:       "备注",
				Required:    false,
				Placeholder: "请输入",
				Validation:  &FormValidation{MinLength: 0, MaxLength: 500},
			},
		},
		SubmitMode: FormSubmitSingle,
		Status:     FormStatusActive,
		CreatedAt:  1719532800000,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded FormDefinitionBody
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Fields) != 2 {
		t.Fatalf("len(Fields) = %d; want 2", len(decoded.Fields))
	}
	if decoded.Fields[0].Type != FormFieldRadio {
		t.Errorf("Fields[0].Type = %q; want %q", decoded.Fields[0].Type, FormFieldRadio)
	}
	if decoded.Fields[1].Validation == nil {
		t.Fatal("Fields[1].Validation is nil")
	}
	if decoded.Fields[1].Validation.MaxLength != 500 {
		t.Errorf("MaxLength = %d; want 500", decoded.Fields[1].Validation.MaxLength)
	}
	if decoded.SubmitMode != FormSubmitSingle {
		t.Errorf("SubmitMode = %q; want %q", decoded.SubmitMode, FormSubmitSingle)
	}
}
