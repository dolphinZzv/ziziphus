// Agent timeline - matches IMCore AgentTimelineBody

export enum AgentStepType {
  Thinking = 'thinking',
  ToolCall = 'toolCall',
  ToolResult = 'toolResult',
  Response = 'response',
}

export enum AgentStepStatus {
  Pending = 'pending',
  Running = 'running',
  Success = 'success',
  Error = 'error',
}

export interface AgentTimelineEntry {
  id: string
  type: AgentStepType
  status: AgentStepStatus
  title: string
  content: string
  timestamp: number
}

export interface AgentTimelineBody {
  parentMsgID: number
  status: AgentStepStatus
  entries: AgentTimelineEntry[]
}
