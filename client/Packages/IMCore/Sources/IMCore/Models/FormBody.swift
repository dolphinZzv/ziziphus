import Foundation

// MARK: - Form definition (ContentType=10)

public struct FormAction: Codable, Sendable, Hashable {
    public var action: String
    public var label: String
    public var style: String  // "primary", "danger", "default"
}

public struct FormField: Codable, Sendable, Hashable {
    public var fieldID: String
    public var type: String   // "text", "radio", "checkbox", etc.
    public var label: String
    public var required: Bool
    public var options: [String]?
    public var placeholder: String?
    public var defaultValue: String?
    public var minLength: Int?
    public var maxLength: Int?

    enum CodingKeys: String, CodingKey {
        case fieldID = "field_id"
        case type, label, required, options, placeholder
        case defaultValue = "default_value"
        case minLength = "min_length"
        case maxLength = "max_length"
    }
}

public struct FormDefinitionBody: Codable, Sendable, Hashable {
    public var formID: String
    public var type: String    // "contact_request" etc.
    public var title: String
    public var description: String?
    public var fromUserID: String?
    public var fromUserName: String?
    public var fromUserAvatar: String?
    public var requestID: Int64
    public var message: String?
    public var fields: [FormField]?
    public var actions: [FormAction]
    public var submitMode: String?  // "single", "multiple"
    public var deadline: Int64?
    public var status: String       // "active", "closed"
    public var createdAt: Int64

    public init(formID: String, type: String, title: String, description: String? = nil,
                fromUserID: String? = nil, fromUserName: String? = nil, fromUserAvatar: String? = nil,
                requestID: Int64, message: String? = nil, fields: [FormField]? = nil,
                actions: [FormAction] = [], submitMode: String? = nil, deadline: Int64? = nil,
                status: String = "active", createdAt: Int64 = 0) {
        self.formID = formID
        self.type = type
        self.title = title
        self.description = description
        self.fromUserID = fromUserID
        self.fromUserName = fromUserName
        self.fromUserAvatar = fromUserAvatar
        self.requestID = requestID
        self.message = message
        self.fields = fields
        self.actions = actions
        self.submitMode = submitMode
        self.deadline = deadline
        self.status = status
        self.createdAt = createdAt
    }

    enum CodingKeys: String, CodingKey {
        case formID = "form_id"
        case type, title, description
        case fromUserID = "from_user_id"
        case fromUserName = "from_user_name"
        case fromUserAvatar = "from_user_avatar"
        case requestID = "request_id"
        case message
        case fields, actions
        case submitMode = "submit_mode"
        case deadline, status
        case createdAt = "created_at"
    }
}

// MARK: - Form response (ContentType=11)

public struct FormAnswer: Codable, Sendable, Hashable {
    public var fieldID: String
    public var value: String

    enum CodingKeys: String, CodingKey {
        case fieldID = "field_id"
        case value
    }
}

public struct FormResponseBody: Codable, Sendable, Hashable {
    public var formMsgID: Int64
    public var requestID: Int64
    public var action: String        // "approve", "reject"
    public var responderID: String
    public var responderName: String
    public var answers: [FormAnswer]?
    public var submittedAt: Int64

    public init(formMsgID: Int64, requestID: Int64, action: String,
                responderID: String, responderName: String, answers: [FormAnswer]? = nil,
                submittedAt: Int64 = 0) {
        self.formMsgID = formMsgID
        self.requestID = requestID
        self.action = action
        self.responderID = responderID
        self.responderName = responderName
        self.answers = answers
        self.submittedAt = submittedAt
    }

    enum CodingKeys: String, CodingKey {
        case formMsgID = "form_msg_id"
        case requestID = "request_id"
        case action
        case responderID = "responder_id"
        case responderName = "responder_name"
        case answers
        case submittedAt = "submitted_at"
    }
}
