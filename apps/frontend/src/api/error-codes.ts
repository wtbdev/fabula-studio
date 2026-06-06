export const ApiErrorCode = {
  Success: 0,
  Unauthorized: 40001,
  ValidationFailed: 40002,
  EmailRegistered: 40003,
  InvalidCredentials: 40004,
  Forbidden: 40005,
  ProjectNotFound: 40401,
  SceneNotFound: 40402,
  SourceTextTooShort: 41001,
  ProjectMissingSourceText: 41002,
  InvalidProjectStatus: 41003,
  InsufficientAiPoints: 50001,
  ServerError: 50000,
  DatabaseFailed: 50002,
  AiGenerationFailed: 51001,
  AiResponseParseFailed: 51002,
  AiGenerationEmpty: 51003,
} as const

export type ApiErrorCode = (typeof ApiErrorCode)[keyof typeof ApiErrorCode]
