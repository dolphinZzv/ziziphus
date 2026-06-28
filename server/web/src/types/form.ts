export enum FormFieldType {
  Text = 'text',
  Textarea = 'textarea',
  Select = 'select',
  Radio = 'radio',
  Checkbox = 'checkbox',
  Date = 'date',
  Time = 'time',
  Number = 'number',
  Rating = 'rating',
}

export enum FormSubmitMode {
  Single = 'single',
  Multiple = 'multiple',
}

export enum FormActionStyle {
  Primary = 'primary',
  Danger = 'danger',
  Default = 'default',
}

export interface FormValidation {
  min_length: number
  max_length: number
  pattern?: string
}

export interface FormField {
  field_id: string
  type: FormFieldType
  label: string
  required: boolean
  options?: string[]
  placeholder?: string
  default_value?: unknown
  validation?: FormValidation
}

export interface FormAction {
  action: string
  label: string
  style: FormActionStyle
}

export interface FormDefinitionBody {
  form_id: string
  type: string // 'contact_request' | future form types
  title: string
  description?: string
  from_user_id?: string
  from_user_name?: string
  from_user_avatar?: string
  request_id: number
  message?: string
  fields?: FormField[]
  actions: FormAction[]
  submit_mode?: FormSubmitMode
  deadline?: number
  status: 'active' | 'closed'
  created_at: number
}

export interface FormAnswer {
  field_id: string
  value: unknown
}

export interface FormResponseBody {
  form_msg_id: number
  request_id: number
  action: string
  responder_id: string
  responder_name: string
  answers?: FormAnswer[]
  submitted_at: number
}
