// Generated from the adopted OpenAPI. Run npm run generate:contract to refresh.
export type RoleCode = "ADMIN" | "ORGANIZER" | "CHECKER" | "READER" | "PARTICIPANT"
export type RoleValue = { "code": RoleCode; "label": string }
export type EventStatusCode = "DRAFT" | "READY" | "CHECKIN_OPEN" | "CHECKIN_PAUSED" | "ENDED" | "ARCHIVED"
export type EventStatusValue = { "code": EventStatusCode; "label": string }
export type PartyStatusCode = "PENDING" | "APPROVED" | "REJECTED" | "CANCELLED"
export type PartyStatusValue = { "code": PartyStatusCode; "label": string }
export type ApprovalModeCode = "IMMEDIATE" | "REVIEW"
export type ApprovalModeValue = { "code": ApprovalModeCode; "label": string }
export type AdmissionProgressCode = "NONE" | "PARTIAL" | "COMPLETE"
export type AdmissionProgressValue = { "code": AdmissionProgressCode; "label": string }
export type CredentialStatusCode = "ACTIVE" | "REVOKED" | "EXPIRED"
export type CredentialStatusValue = { "code": CredentialStatusCode; "label": string }
export type SourceTypeCode = "BUILTIN" | "FILE" | "FEISHU" | "TENCENT" | "WALK_IN"
export type SourceTypeValue = { "code": SourceTypeCode; "label": string }
export type SourceEligibilityCode = "ELIGIBLE" | "REVIEW_REQUIRED"
export type SourceEligibilityValue = { "code": SourceEligibilityCode; "label": string }
export type ProviderCode = "FEISHU" | "TENCENT" | "EMAIL" | "SMS"
export type ProviderValue = { "code": ProviderCode; "label": string }
export type AccountKindCode = "PERSONAL" | "ORGANIZATION"
export type AccountKindValue = { "code": AccountKindCode; "label": string }
export type ConnectionStatusCode = "UNAUTHORIZED" | "AUTHORIZED" | "EXPIRED" | "REVOKED" | "UNAVAILABLE"
export type ConnectionStatusValue = { "code": ConnectionStatusCode; "label": string }
export type JobStatusCode = "PENDING" | "RUNNING" | "SUCCEEDED" | "FAILED" | "EXPIRED"
export type JobStatusValue = { "code": JobStatusCode; "label": string }
export type JobKindCode = "PULL" | "EMAIL" | "EXPORT" | "CLEANUP"
export type JobKindValue = { "code": JobKindCode; "label": string }
export type BatchStatusCode = "UPLOADED" | "PREVIEWED" | "COMMITTED" | "FAILED" | "EXPIRED"
export type BatchStatusValue = { "code": BatchStatusCode; "label": string }
export type RowStatusCode = "VALID" | "ERROR" | "DUPLICATE_SUSPECT" | "ALREADY_LINKED" | "EXCLUDED" | "COMMITTED"
export type RowStatusValue = { "code": RowStatusCode; "label": string }
export type DeviceModeCode = "STAFF" | "KIOSK"
export type DeviceModeValue = { "code": DeviceModeCode; "label": string }
export type DeviceStatusCode = "UNCLAIMED" | "ACTIVE" | "REVOKED" | "EXPIRED"
export type DeviceStatusValue = { "code": DeviceStatusCode; "label": string }
export type OperationStatusCode = "IN_PROGRESS" | "SUCCEEDED" | "REJECTED"
export type OperationStatusValue = { "code": OperationStatusCode; "label": string }
export type ReviewKindCode = "INITIAL" | "INCREASE"
export type ReviewKindValue = { "code": ReviewKindCode; "label": string }
export type ReviewResultCode = "PENDING" | "APPROVED" | "REJECTED" | "CANCELLED"
export type ReviewResultValue = { "code": ReviewResultCode; "label": string }
export type PartyFieldCode = "name" | "phone" | "email" | "companionCount"
export type PartyFieldValue = { "code": PartyFieldCode; "label": string }
export type EventActionCode = "PUBLISH" | "OPEN" | "PAUSE" | "RESUME" | "END"
export type EventActionValue = { "code": EventActionCode; "label": string }
export type ExportFormatCode = "CSV" | "XLSX"
export type ExportFormatValue = { "code": ExportFormatCode; "label": string }
export type RecoveryChannelCode = "EMAIL" | "SMS"
export type RecoveryChannelValue = { "code": RecoveryChannelCode; "label": string }
export type CountModeCode = "COMPANIONS" | "TOTAL"
export type CountModeValue = { "code": CountModeCode; "label": string }
export type DuplicateResolutionCode = "LINK_EXISTING" | "KEEP_SEPARATE" | "EXCLUDE"
export type DuplicateResolutionValue = { "code": DuplicateResolutionCode; "label": string }
export type DecisionCode = "APPROVE" | "REJECT"
export type DecisionValue = { "code": DecisionCode; "label": string }
export type RetryModeCode = "NONE" | "SAME_OPERATION" | "NEW_AFTER_CONFIRM"
export type RetryModeValue = { "code": RetryModeCode; "label": string }
export type Empty = {  }
export type FieldRule = { "enabled": boolean; "required": boolean }
export type FieldRules = { "name": FieldRule; "phone": FieldRule; "email": FieldRule; "companionCount": FieldRule }
export type PartyInput = { "name"?: string; "phone"?: string; "email"?: string; "companionCount"?: number }
export type FieldError = { "field": string; "code": string; "message": string }
export type ErrorData = { "operationId"?: string; "fieldErrors"?: Array<FieldError>; "retryMode"?: RetryModeCode; "retryAfterSeconds"?: number }
export type ErrorEnvelope = { "code": string; "message": string; "data": ErrorData; "requestId": string }
export type CsrfView = { "csrfToken": string; "expiresAt": string; "clientContextId": string }
export type SessionView = { "accountId": string; "username": string; "displayName": string; "roles": Array<RoleValue>; "csrfToken": string; "expiresAt": string; "mustChangePassword": boolean }
export type AccountView = { "accountId": string; "username": string; "displayName": string; "enabled": boolean; "roles": Array<RoleValue>; "version": number }
export type GrantView = { "grantId": string; "accountId": string; "eventId": string; "role": RoleValue; "allowedFields": Array<PartyFieldValue>; "allowedActions": Array<string>; "recipient"?: string; "purpose"?: string; "retentionNote"?: string; "expiresAt": string; "version": number; "accountDisplayName": string; "username": string }
export type EventView = { "eventId": string; "title": string; "venue"?: string; "startsAt": string; "endsAt": string; "timeZone": string; "registrationStartsAt": string; "registrationEndsAt": string; "capacity": number; "allocatedCount": number; "status": EventStatusValue; "version": number }
export type EventPolicy = { "fields": FieldRules; "approvalMode": ApprovalModeValue; "phoneVerificationEnabled": boolean; "selfEditEnabled": boolean; "selfCancelEnabled": boolean; "selfServiceEndsAt"?: string; "retentionDays": number; "version": number }
export type EventPublicView = { "eventId": string; "title": string; "venue"?: string; "startsAt": string; "endsAt": string; "timeZone": string; "fields": FieldRules; "approvalMode": ApprovalModeValue; "phoneVerificationEnabled": boolean; "registrationAvailable": boolean; "notice": string; "noticeVersion": string; "retentionDays": number }
export type PartyView = { "partyId": string; "partyReference": string; "eventId": string; "fields": PartyInput; "requestedSize": number; "approvedSize": number; "pendingIncrease": number; "netAdmitted": number; "remainingCount": number; "status": PartyStatusValue; "progress": AdmissionProgressValue; "sourceType": SourceTypeValue; "version": number; "createdAt": string; "updatedAt": string; "hasAdmissionHistory": boolean }
export type PartyAccessBundle = { "queryToken": string; "changeToken"?: string; "expiresAt": string }
export type SubmissionView = { "party": PartyView; "access": PartyAccessBundle; "operationId": string; "replayed": boolean }
export type ReissueView = { "party": PartyView; "credential": CredentialView; "access"?: PartyAccessBundle; "operationId": string; "replayed": boolean }
export type PartyStateView = { "partyId": string; "partyReference": string; "eventId": string; "fields": PartyInput; "status": PartyStatusValue; "approvedSize": number; "pendingIncrease": number; "netAdmitted": number; "remainingCount": number; "progress": AdmissionProgressValue; "canChange": boolean; "canCancel": boolean; "version": number; "expiresAt": string }
export type CredentialView = { "partyId": string; "status": CredentialStatusValue; "credentialVersion": number; "ticketToken"?: string; "expiresAt": string }
export type ReviewView = { "reviewId": string; "partyId": string; "eventId": string; "kind": ReviewKindValue; "requestedSize": number; "approvedSize": number; "requestedIncrease": number; "result": ReviewResultValue; "reason"?: string; "version": number; "createdAt": string }
export type GatePreview = { "scanContextToken": string; "eventId": string; "approvedSize": number; "netAdmitted": number; "remainingCount": number; "partyVersion": number; "progress": AdmissionProgressValue; "expiresAt": string }
export type GateCandidate = { "partyId": string; "partyReference": string; "displayNameMasked"?: string; "phoneSuffix"?: string; "approvedSize": number; "netAdmitted": number; "remainingCount": number; "progress": AdmissionProgressValue }
export type GateCandidatePage = { "items": Array<GateCandidate>; "total": number; "page": number; "pageSize": number }
export type GateResult = { "operationId": string; "recordId"?: string; "status": OperationStatusValue; "requestedCount": number; "committedCount": number; "remainingCount": number; "committedAt"?: string; "replayed": boolean; "showAsNewAdmission": boolean }
export type AdmissionView = { "recordId": string; "partyId": string; "eventId": string; "operationId": string; "deviceId": string; "actorId"?: string; "admittedCount": number; "correctedCount": number; "committedAt": string }
export type CorrectionView = { "correctionId": string; "originalRecordId": string; "partyId": string; "correctedCount": number; "reason": string; "createdAt": string; "operationId": string }
export type OperationView = { "operationId": string; "status": OperationStatusValue; "action": string; "resourceId"?: string; "committedAt"?: string; "resultCode": string; "replayed": boolean }
export type DeviceView = { "deviceId": string; "eventId": string; "name": string; "mode": DeviceModeValue; "status": DeviceStatusValue; "expiresAt": string; "lastSeenAt"?: string; "version": number }
export type DeviceRegistration = { "device": DeviceView; "pairingCode": string; "pairingExpiresAt": string }
export type DeviceSessionView = { "device": DeviceView; "csrfToken": string; "expiresAt": string; "eventStatus": EventStatusValue }
export type ConnectionView = { "connectionId": string; "provider": ProviderValue; "accountKind": AccountKindValue; "displayName": string; "status": ConnectionStatusValue; "allowedEventIds": Array<string>; "credentialMask"?: string; "expiresAt"?: string; "version": number }
export type AuthorizationView = { "authorizationUrl": string; "expiresAt": string }
export type ResourceView = { "resourceId": string; "title": string; "kind": string; "readable": boolean }
export type ResourcePage = { "items": Array<ResourceView>; "nextCursor"?: string }
export type FieldMapping = { "nameColumn"?: string; "phoneColumn"?: string; "emailColumn"?: string; "countColumn"?: string; "countMode": CountModeCode }
export type SourceView = { "sourceId": string; "eventId": string; "connectionId": string; "resourceId": string; "provider": ProviderValue; "mapping": FieldMappingView; "eligibility": SourceEligibilityValue; "version": number }
export type SourceListView = { "items": Array<SourceView>; "availableConnections": Array<SourceConnectionOption> }
export type BatchView = { "batchId": string; "eventId": string; "sourceId"?: string; "fileDigest"?: string; "status": BatchStatusValue; "eligibility"?: SourceEligibilityValue; "mapping"?: FieldMappingView; "previewVersion": number; "validRows": number; "errorRows": number; "suspectRows": number; "policyVersion"?: number; "expiresAt": string; "version": number; "availableColumns": Array<SourceColumnOption> }
export type ImportRowView = { "rowId": string; "sourceRecordId"?: string; "lineNumber": number; "fields": PartyInput; "requestedSize": number; "status": RowStatusValue; "errors": Array<FieldError>; "linkedPartyId"?: string }
export type BatchCommitView = { "batchId": string; "operationId": string; "createdCount": number; "linkedCount": number; "allocatedPeople": number; "replayed": boolean }
export type JobView = { "jobId": string; "eventId"?: string; "kind": JobKindValue; "status": JobStatusValue; "attempts": number; "createdAt": string; "completedAt"?: string; "resultRef"?: string; "errorCode"?: string; "expiresAt"?: string; "version": number }
export type ExportView = { "exportId": string; "eventId": string; "status": JobStatusValue; "format": ExportFormatValue; "columns": Array<string>; "rowCount"?: number; "expiresAt": string; "version": number }
export type StatisticsView = { "eventId": string; "asOf": string; "applicationGroups": number; "pendingGroups": number; "approvedGroups": number; "allocatedPeople": number; "grossAdmissions": number; "correctedPeople": number; "netAdmissions": number; "unusedPlaces": number; "walkInGroups": number; "onsiteAddedPlaces": number; "admissionRate": number | null }
export type AuditView = { "auditId": string; "eventId"?: string; "action": string; "actorId"?: string; "deviceId"?: string; "resourceId"?: string; "reason"?: string; "occurredAt": string }
export type DeliveryView = { "deliveryId": string; "eventId": string; "exportId": string; "recipient": string; "purpose": string; "columns": Array<string>; "retentionNote": string; "createdAt": string; "version": number }
export type CleanupView = { "cleanupId": string; "eventId": string; "dueAt": string; "status": JobStatusValue; "processedCount": number; "failedCount": number; "completedAt"?: string; "version": number; "scopes": Array<CleanupScopeResult> }
export type ChallengeView = { "challengeId": string; "expiresAt": string; "retryAfterSeconds": number }
export type VerificationView = { "verificationProof": string; "expiresAt": string }
export type OptionsView = { "items": Array<EventView>; "nextCursor"?: string }
export type PartyFilter = { "status"?: PartyStatusCode; "progress"?: AdmissionProgressCode; "sourceType"?: SourceTypeCode; "createdFrom"?: string; "createdTo"?: string }
export type EventPolicyInput = { "fields": FieldRules; "approvalMode": ApprovalModeCode; "phoneVerificationEnabled": boolean; "selfEditEnabled": boolean; "selfCancelEnabled": boolean; "selfServiceEndsAt"?: string; "retentionDays": number; "version": number }
export type FieldMappingView = { "nameColumn"?: string; "phoneColumn"?: string; "emailColumn"?: string; "countColumn"?: string; "countMode": CountModeValue }
export type AccountPage = { "items": Array<AccountView>; "total": number; "page": number; "pageSize": number }
export type GrantPage = { "items": Array<GrantView>; "total": number; "page": number; "pageSize": number }
export type EventPage = { "items": Array<EventView>; "total": number; "page": number; "pageSize": number }
export type PartyPage = { "items": Array<PartyView>; "total": number; "page": number; "pageSize": number; "asOf": string }
export type ReviewPage = { "items": Array<ReviewView>; "total": number; "page": number; "pageSize": number }
export type ImportRowPage = { "items": Array<ImportRowView>; "total": number; "page": number; "pageSize": number }
export type AdmissionPage = { "items": Array<AdmissionView>; "total": number; "page": number; "pageSize": number }
export type AuditPage = { "items": Array<AuditView>; "total": number; "page": number; "pageSize": number }
export type JobPage = { "items": Array<JobView>; "total": number; "page": number; "pageSize": number }
export type DeliveryPage = { "items": Array<DeliveryView>; "total": number; "page": number; "pageSize": number }
export type ConnectionPage = { "items": Array<ConnectionView>; "total": number; "page": number; "pageSize": number }
export type DevicePage = { "items": Array<DeviceView>; "total": number; "page": number; "pageSize": number }
export type SessionBootstrapRequest = {  }
export type SessionBootstrapResponse = { "code": string; "message": string; "data": CsrfView; "requestId": string }
export type LoginRequest = { "operationId": string; "username": string; "password": string }
export type LoginResponse = { "code": string; "message": string; "data": SessionView; "requestId": string }
export type LogoutRequest = { "operationId": string }
export type LogoutResponse = { "code": string; "message": string; "data": Empty; "requestId": string }
export type SessionMeRequest = {  }
export type SessionMeResponse = { "code": string; "message": string; "data": SessionView; "requestId": string }
export type AccountListRequest = { "page"?: number; "pageSize"?: number; "query"?: string }
export type AccountListResponse = { "code": string; "message": string; "data": AccountPage; "requestId": string }
export type AccountCreateRequest = { "operationId": string; "username": string; "displayName": string; "temporaryPassword": string; "roles": Array<"ADMIN" | "ORGANIZER" | "CHECKER" | "READER"> }
export type AccountCreateResponse = { "code": string; "message": string; "data": AccountView; "requestId": string }
export type AccountUpdateRequest = { "operationId": string; "accountId": string; "version": number; "displayName": string; "enabled": boolean; "roles": Array<"ADMIN" | "ORGANIZER" | "CHECKER" | "READER"> }
export type AccountUpdateResponse = { "code": string; "message": string; "data": AccountView; "requestId": string }
export type PasswordChangeRequest = { "operationId": string; "oldPassword": string; "newPassword": string }
export type PasswordChangeResponse = { "code": string; "message": string; "data": Empty; "requestId": string }
export type GrantListRequest = { "eventId": string; "page"?: number; "pageSize"?: number }
export type GrantListResponse = { "code": string; "message": string; "data": GrantPage; "requestId": string }
export type GrantSaveRequest = { "operationId": string; "eventId": string; "accountId": string; "role": "ORGANIZER" | "CHECKER" | "READER"; "allowedFields": Array<PartyFieldCode>; "allowedActions": Array<string>; "recipient"?: string; "purpose"?: string; "retentionNote"?: string; "expiresAt": string; "grantId"?: string; "version"?: number }
export type GrantSaveResponse = { "code": string; "message": string; "data": GrantView; "requestId": string }
export type GrantRevokeRequest = { "operationId": string; "eventId": string; "grantId": string; "version": number }
export type GrantRevokeResponse = { "code": string; "message": string; "data": Empty; "requestId": string }
export type EventListRequest = { "page"?: number; "pageSize"?: number; "status"?: EventStatusCode; "query"?: string }
export type EventListResponse = { "code": string; "message": string; "data": EventPage; "requestId": string }
export type EventDetailRequest = { "eventId": string }
export type EventDetailResponse = { "code": string; "message": string; "data": EventView; "requestId": string }
export type EventCreateRequest = { "operationId": string; "title": string; "venue"?: string; "startsAt": string; "endsAt": string; "timeZone": string; "registrationStartsAt": string; "registrationEndsAt": string; "capacity": number }
export type EventCreateResponse = { "code": string; "message": string; "data": EventView; "requestId": string }
export type EventUpdateRequest = { "operationId": string; "eventId": string; "version": number; "title": string; "venue"?: string; "startsAt": string; "endsAt": string; "timeZone": string; "registrationStartsAt": string; "registrationEndsAt": string; "capacity": number }
export type EventUpdateResponse = { "code": string; "message": string; "data": EventView; "requestId": string }
export type EventTransitionRequest = { "operationId": string; "eventId": string; "version": number; "action": EventActionCode; "reason"?: string }
export type EventTransitionResponse = { "code": string; "message": string; "data": EventView; "requestId": string }
export type PolicyGetRequest = { "eventId": string }
export type PolicyGetResponse = { "code": string; "message": string; "data": EventPolicy; "requestId": string }
export type PolicyUpdateRequest = { "operationId": string; "eventId": string; "policy": EventPolicyInput }
export type PolicyUpdateResponse = { "code": string; "message": string; "data": EventPolicy; "requestId": string }
export type EventOptionsRequest = { "cursor"?: string }
export type EventOptionsResponse = { "code": string; "message": string; "data": OptionsView; "requestId": string }
export type PublicEventRequest = { "eventId": string }
export type PublicEventResponse = { "code": string; "message": string; "data": EventPublicView; "requestId": string }
export type SubmitRegistrationRequest = { "operationId": string; "eventId": string; "fields": PartyInput; "phoneVerificationProof"?: string; "acceptedNoticeVersion": string }
export type SubmitRegistrationResponse = { "code": string; "message": string; "data": SubmissionView; "requestId": string }
export type PartyExchangeRequest = { "operationId": string; "queryToken": string; "changeToken"?: string }
export type PartyExchangeResponse = { "code": string; "message": string; "data": CsrfView; "requestId": string }
export type MyPartyRequest = {  }
export type MyPartyResponse = { "code": string; "message": string; "data": PartyStateView; "requestId": string }
export type MyCredentialRequest = {  }
export type MyCredentialResponse = { "code": string; "message": string; "data": CredentialView; "requestId": string }
export type MyPartyChangeRequest = { "operationId": string; "version": number; "companionCount": number }
export type MyPartyChangeResponse = { "code": string; "message": string; "data": PartyStateView; "requestId": string }
export type MyPartyCancelRequest = { "operationId": string; "version": number }
export type MyPartyCancelResponse = { "code": string; "message": string; "data": PartyStateView; "requestId": string }
export type MyOperationRequest = { "operationId": string; "action": string }
export type MyOperationResponse = { "code": string; "message": string; "data": OperationView; "requestId": string }
export type SmsStartRequest = { "operationId": string; "eventId": string; "phone": string }
export type SmsStartResponse = { "code": string; "message": string; "data": ChallengeView; "requestId": string }
export type SmsVerifyRequest = { "operationId": string; "eventId": string; "challengeId": string; "code": string }
export type SmsVerifyResponse = { "code": string; "message": string; "data": VerificationView; "requestId": string }
export type RecoveryStartRequest = { "operationId": string; "eventId": string; "partyReference": string; "channel": RecoveryChannelCode; "contact": string }
export type RecoveryStartResponse = { "code": string; "message": string; "data": ChallengeView; "requestId": string }
export type RecoveryVerifyRequest = { "operationId": string; "challengeId": string; "code": string }
export type RecoveryVerifyResponse = { "code": string; "message": string; "data": PartyAccessBundle; "requestId": string }
export type PartyListRequest = { "eventId": string; "page"?: number; "pageSize"?: number; "status"?: PartyStatusCode; "progress"?: AdmissionProgressCode; "sourceType"?: SourceTypeCode; "query"?: string; "createdFrom"?: string; "createdTo"?: string; "hasPendingIncrease"?: boolean }
export type PartyListResponse = { "code": string; "message": string; "data": PartyPage; "requestId": string }
export type PartyDetailRequest = { "eventId": string; "partyId": string }
export type PartyDetailResponse = { "code": string; "message": string; "data": PartyView; "requestId": string }
export type PartyChangeRequest = { "operationId": string; "eventId": string; "partyId": string; "version": number; "companionCount": number; "reason": string }
export type PartyChangeResponse = { "code": string; "message": string; "data": PartyView; "requestId": string }
export type PartyCancelRequest = { "operationId": string; "eventId": string; "partyId": string; "version": number; "reason": string }
export type PartyCancelResponse = { "code": string; "message": string; "data": PartyView; "requestId": string }
export type ReviewListRequest = { "eventId": string; "page"?: number; "pageSize"?: number; "kind"?: ReviewKindCode; "result"?: ReviewResultCode }
export type ReviewListResponse = { "code": string; "message": string; "data": ReviewPage; "requestId": string }
export type ReviewDecideRequest = { "operationId": string; "eventId": string; "reviewId": string; "version": number; "decision": DecisionCode; "reason": string }
export type ReviewDecideResponse = { "code": string; "message": string; "data": ReviewView; "requestId": string }
export type CredentialReissueRequest = { "operationId": string; "eventId": string; "partyId": string; "version": number; "reason": string; "rotateAccess": boolean }
export type CredentialReissueResponse = { "code": string; "message": string; "data": ReissueView; "requestId": string }
export type CredentialRevokeRequest = { "operationId": string; "eventId": string; "partyId": string; "version": number; "reason": string }
export type CredentialRevokeResponse = { "code": string; "message": string; "data": PartyView; "requestId": string }
export type DeviceRegisterRequest = { "operationId": string; "eventId": string; "name": string; "mode": DeviceModeCode }
export type DeviceRegisterResponse = { "code": string; "message": string; "data": DeviceRegistration; "requestId": string }
export type DeviceClaimRequest = { "operationId": string; "deviceId": string; "pairingCode": string }
export type DeviceClaimResponse = { "code": string; "message": string; "data": DeviceSessionView; "requestId": string }
export type DeviceDetailRequest = {  }
export type DeviceDetailResponse = { "code": string; "message": string; "data": DeviceSessionView; "requestId": string }
export type DeviceListRequest = { "eventId": string; "page"?: number; "pageSize"?: number }
export type DeviceListResponse = { "code": string; "message": string; "data": DevicePage; "requestId": string }
export type DeviceRevokeRequest = { "operationId": string; "eventId": string; "deviceId": string; "version": number; "reason": string }
export type DeviceRevokeResponse = { "code": string; "message": string; "data": Empty; "requestId": string }
export type TicketResolveRequest = { "eventId": string; "ticketToken": string }
export type TicketResolveResponse = { "code": string; "message": string; "data": GatePreview; "requestId": string }
export type GateCommitRequest = { "operationId": string; "eventId": string; "scanContextToken": string; "expectedPartyVersion": number; "count": number }
export type GateCommitResponse = { "code": string; "message": string; "data": GateResult; "requestId": string }
export type GateLookupRequest = { "eventId": string; "query": string }
export type GateLookupResponse = { "code": string; "message": string; "data": GateCandidatePage; "requestId": string }
export type GateOperationRequest = { "operationId": string; "scanContextToken"?: string }
export type GateOperationResponse = { "code": string; "message": string; "data": GateResult; "requestId": string }
export type GateRecordsRequest = { "eventId": string; "page"?: number; "pageSize"?: number; "deviceId"?: string; "partyId"?: string }
export type GateRecordsResponse = { "code": string; "message": string; "data": AdmissionPage; "requestId": string }
export type GateCorrectRequest = { "operationId": string; "eventId": string; "recordId": string; "partyVersion": number; "correctedCount": number; "reason": string }
export type GateCorrectResponse = { "code": string; "message": string; "data": CorrectionView; "requestId": string }
export type GateWalkInRequest = { "operationId": string; "eventId": string; "fields": PartyInput; "reason": string; "existingPendingPartyId"?: string; "partyVersion"?: number; "phoneVerificationProof"?: string }
export type GateWalkInResponse = { "code": string; "message": string; "data": PartyView; "requestId": string }
export type GateIncreaseRequest = { "operationId": string; "eventId": string; "partyId": string; "version": number; "additionalCount": number; "reason": string }
export type GateIncreaseResponse = { "code": string; "message": string; "data": PartyView; "requestId": string }
export type ConnectionListRequest = { "page"?: number; "pageSize"?: number; "provider"?: "FEISHU" | "TENCENT" | "EMAIL" | "SMS" }
export type ConnectionListResponse = { "code": string; "message": string; "data": ConnectionPage; "requestId": string }
export type ConnectionSaveRequest = { "operationId": string; "connectionId"?: string; "version"?: number; "provider": "FEISHU" | "TENCENT" | "EMAIL" | "SMS"; "accountKind": AccountKindCode; "displayName": string; "allowedEventIds": Array<string>; "clientId"?: string; "clientSecret"?: string }
export type ConnectionSaveResponse = { "code": string; "message": string; "data": ConnectionView; "requestId": string }
export type ConnectionAuthorizeRequest = { "operationId": string; "connectionId": string; "version": number }
export type ConnectionAuthorizeResponse = { "code": string; "message": string; "data": AuthorizationView; "requestId": string }
export type ConnectionCallbackRequest = { "state": string; "code": string }
export type ConnectionCallbackResponse = { "code": string; "message": string; "data": ConnectionView; "requestId": string }
export type ConnectionRevokeRequest = { "operationId": string; "connectionId": string; "version": number }
export type ConnectionRevokeResponse = { "code": string; "message": string; "data": ConnectionView; "requestId": string }
export type ResourceListRequest = { "connectionId": string; "eventId"?: string; "cursor"?: string }
export type ResourceListResponse = { "code": string; "message": string; "data": ResourcePage; "requestId": string }
export type SourceSaveRequest = { "operationId": string; "eventId": string; "sourceId"?: string; "version"?: number; "connectionId": string; "resourceId": string; "mapping": FieldMapping; "eligibility": SourceEligibilityCode }
export type SourceSaveResponse = { "code": string; "message": string; "data": SourceView; "requestId": string }
export type SourceListRequest = { "eventId": string }
export type SourceListResponse = { "code": string; "message": string; "data": SourceListView; "requestId": string }
export type SourcePullRequest = { "operationId": string; "eventId": string; "sourceId": string; "version": number }
export type SourcePullResponse = { "code": string; "message": string; "data": JobView; "requestId": string }
export type BatchUploadRequest = { "operationId": string; "eventId": string; "file": string; "sourceLabel": string }
export type BatchUploadResponse = { "code": string; "message": string; "data": BatchView; "requestId": string }
export type BatchDetailRequest = { "eventId": string; "batchId": string }
export type BatchDetailResponse = { "code": string; "message": string; "data": BatchView; "requestId": string }
export type BatchPreviewRequest = { "operationId": string; "eventId": string; "batchId": string; "version": number; "mapping": FieldMapping; "eligibility": SourceEligibilityCode }
export type BatchPreviewResponse = { "code": string; "message": string; "data": BatchView; "requestId": string }
export type BatchRowsRequest = { "eventId": string; "batchId": string; "page"?: number; "pageSize"?: number; "status"?: RowStatusCode }
export type BatchRowsResponse = { "code": string; "message": string; "data": ImportRowPage; "requestId": string }
export type DuplicateResolveRequest = { "operationId": string; "eventId": string; "batchId": string; "rowId": string; "previewVersion": number; "resolution": DuplicateResolutionCode; "existingPartyId"?: string; "reason": string }
export type DuplicateResolveResponse = { "code": string; "message": string; "data": BatchView; "requestId": string }
export type BatchCommitRequest = { "operationId": string; "eventId": string; "batchId": string; "previewVersion": number; "selectedRowIds": Array<string> }
export type BatchCommitResponse = { "code": string; "message": string; "data": BatchCommitView; "requestId": string }
export type JobListRequest = { "eventId"?: string; "page"?: number; "pageSize"?: number; "kind"?: JobKindCode }
export type JobListResponse = { "code": string; "message": string; "data": JobPage; "requestId": string }
export type JobDetailRequest = { "jobId": string; "eventId"?: string }
export type JobDetailResponse = { "code": string; "message": string; "data": JobView; "requestId": string }
export type JobRetryRequest = { "operationId": string; "jobId": string; "eventId"?: string; "version": number }
export type JobRetryResponse = { "code": string; "message": string; "data": JobView; "requestId": string }
export type EmailResendRequest = { "operationId": string; "eventId": string; "partyId": string; "reason": string }
export type EmailResendResponse = { "code": string; "message": string; "data": JobView; "requestId": string }
export type StatisticsRequest = { "eventId": string }
export type StatisticsResponse = { "code": string; "message": string; "data": StatisticsView; "requestId": string }
export type ExportCreateRequest = { "operationId": string; "eventId": string; "format": ExportFormatCode; "columns": Array<string>; "purpose": string; "filter"?: PartyFilter }
export type ExportCreateResponse = { "code": string; "message": string; "data": ExportView; "requestId": string }
export type ExportDetailRequest = { "eventId": string; "exportId": string }
export type ExportDetailResponse = { "code": string; "message": string; "data": ExportView; "requestId": string }
export type ExportDownloadRequest = { "eventId": string; "exportId": string }
export type ExportDownloadResponse = { "code": string; "message": string; "data": Empty; "requestId": string }
export type DeliveryListRequest = { "eventId": string; "page"?: number; "pageSize"?: number }
export type DeliveryListResponse = { "code": string; "message": string; "data": DeliveryPage; "requestId": string }
export type DeliveryRecordRequest = { "operationId": string; "eventId": string; "exportId": string; "recipient": string; "purpose": string; "columns": Array<string>; "retentionNote": string }
export type DeliveryRecordResponse = { "code": string; "message": string; "data": DeliveryView; "requestId": string }
export type CleanupDetailRequest = { "eventId": string }
export type CleanupDetailResponse = { "code": string; "message": string; "data": CleanupView; "requestId": string }
export type CleanupRetryRequest = { "operationId": string; "eventId": string; "cleanupId": string; "version": number }
export type CleanupRetryResponse = { "code": string; "message": string; "data": JobView; "requestId": string }
export type AuditListRequest = { "eventId": string; "page"?: number; "pageSize"?: number; "action"?: string; "from"?: string; "to"?: string }
export type AuditListResponse = { "code": string; "message": string; "data": AuditPage; "requestId": string }
export type OperationResultRequest = { "eventId": string; "operationId": string; "action": string }
export type OperationResultResponse = { "code": string; "message": string; "data": OperationView; "requestId": string }
export type MyGrantListRequest = { "eventId"?: string; "page"?: number; "pageSize"?: number }
export type MyGrantListResponse = { "code": string; "message": string; "data": GrantPage; "requestId": string }
export type SourceColumnOption = { "code": string; "label": string }
export type ResourceFieldsView = { "resourceId": string; "availableColumns": Array<SourceColumnOption> }
export type CleanupScopeCode = "PRIVATE_ACCESS" | "LIVE_ROWS" | "TEMP_FILES" | "EXPORT_FILES" | "TASK_PAYLOADS" | "BACKUPS"
export type CleanupScopeValue = { "code": CleanupScopeCode; "label": string }
export type CleanupScopeResult = { "scope": CleanupScopeValue; "status": JobStatusValue; "processedCount": number; "failedCount": number; "errorCode"?: string }
export type ResourceFieldsRequest = { "connectionId": string; "eventId"?: string; "resourceId": string }
export type ResourceFieldsResponse = { "code": string; "message": string; "data": ResourceFieldsView; "requestId": string }
export type GrantEventOption = { "eventId": string; "title": string; "startsAt": string }
export type GrantEventOptionsView = { "items": Array<GrantEventOption>; "nextCursor"?: string }
export type GrantEventOptionsRequest = { "query"?: string; "cursor"?: string }
export type GrantEventOptionsResponse = { "code": string; "message": string; "data": GrantEventOptionsView; "requestId": string }
export type RegistrationResultRequest = { "eventId": string; "operationId": string }
export type RegistrationResultResponse = { "code": string; "message": string; "data": SubmissionView; "requestId": string }
export type SourceConnectionOption = { "connectionId": string; "displayName": string; "provider": ProviderValue; "accountKind": AccountKindValue; "status": ConnectionStatusValue }
export interface OperationMap {
  SessionBootstrap: { request: SessionBootstrapRequest; data: CsrfView }
  Login: { request: LoginRequest; data: SessionView }
  Logout: { request: LogoutRequest; data: Empty }
  SessionMe: { request: SessionMeRequest; data: SessionView }
  AccountList: { request: AccountListRequest; data: AccountPage }
  AccountCreate: { request: AccountCreateRequest; data: AccountView }
  AccountUpdate: { request: AccountUpdateRequest; data: AccountView }
  PasswordChange: { request: PasswordChangeRequest; data: Empty }
  GrantList: { request: GrantListRequest; data: GrantPage }
  GrantSave: { request: GrantSaveRequest; data: GrantView }
  GrantRevoke: { request: GrantRevokeRequest; data: Empty }
  EventList: { request: EventListRequest; data: EventPage }
  EventDetail: { request: EventDetailRequest; data: EventView }
  EventCreate: { request: EventCreateRequest; data: EventView }
  EventUpdate: { request: EventUpdateRequest; data: EventView }
  EventTransition: { request: EventTransitionRequest; data: EventView }
  PolicyGet: { request: PolicyGetRequest; data: EventPolicy }
  PolicyUpdate: { request: PolicyUpdateRequest; data: EventPolicy }
  EventOptions: { request: EventOptionsRequest; data: OptionsView }
  PublicEvent: { request: PublicEventRequest; data: EventPublicView }
  SubmitRegistration: { request: SubmitRegistrationRequest; data: SubmissionView }
  PartyExchange: { request: PartyExchangeRequest; data: CsrfView }
  MyParty: { request: MyPartyRequest; data: PartyStateView }
  MyCredential: { request: MyCredentialRequest; data: CredentialView }
  MyPartyChange: { request: MyPartyChangeRequest; data: PartyStateView }
  MyPartyCancel: { request: MyPartyCancelRequest; data: PartyStateView }
  MyOperation: { request: MyOperationRequest; data: OperationView }
  SmsStart: { request: SmsStartRequest; data: ChallengeView }
  SmsVerify: { request: SmsVerifyRequest; data: VerificationView }
  RecoveryStart: { request: RecoveryStartRequest; data: ChallengeView }
  RecoveryVerify: { request: RecoveryVerifyRequest; data: PartyAccessBundle }
  PartyList: { request: PartyListRequest; data: PartyPage }
  PartyDetail: { request: PartyDetailRequest; data: PartyView }
  PartyChange: { request: PartyChangeRequest; data: PartyView }
  PartyCancel: { request: PartyCancelRequest; data: PartyView }
  ReviewList: { request: ReviewListRequest; data: ReviewPage }
  ReviewDecide: { request: ReviewDecideRequest; data: ReviewView }
  CredentialReissue: { request: CredentialReissueRequest; data: ReissueView }
  CredentialRevoke: { request: CredentialRevokeRequest; data: PartyView }
  DeviceRegister: { request: DeviceRegisterRequest; data: DeviceRegistration }
  DeviceClaim: { request: DeviceClaimRequest; data: DeviceSessionView }
  DeviceDetail: { request: DeviceDetailRequest; data: DeviceSessionView }
  DeviceList: { request: DeviceListRequest; data: DevicePage }
  DeviceRevoke: { request: DeviceRevokeRequest; data: Empty }
  TicketResolve: { request: TicketResolveRequest; data: GatePreview }
  GateCommit: { request: GateCommitRequest; data: GateResult }
  GateLookup: { request: GateLookupRequest; data: GateCandidatePage }
  GateOperation: { request: GateOperationRequest; data: GateResult }
  GateRecords: { request: GateRecordsRequest; data: AdmissionPage }
  GateCorrect: { request: GateCorrectRequest; data: CorrectionView }
  GateWalkIn: { request: GateWalkInRequest; data: PartyView }
  GateIncrease: { request: GateIncreaseRequest; data: PartyView }
  ConnectionList: { request: ConnectionListRequest; data: ConnectionPage }
  ConnectionSave: { request: ConnectionSaveRequest; data: ConnectionView }
  ConnectionAuthorize: { request: ConnectionAuthorizeRequest; data: AuthorizationView }
  ConnectionCallback: { request: ConnectionCallbackRequest; data: ConnectionView }
  ConnectionRevoke: { request: ConnectionRevokeRequest; data: ConnectionView }
  ResourceList: { request: ResourceListRequest; data: ResourcePage }
  SourceSave: { request: SourceSaveRequest; data: SourceView }
  SourceList: { request: SourceListRequest; data: SourceListView }
  SourcePull: { request: SourcePullRequest; data: JobView }
  BatchUpload: { request: BatchUploadRequest; data: BatchView }
  BatchDetail: { request: BatchDetailRequest; data: BatchView }
  BatchPreview: { request: BatchPreviewRequest; data: BatchView }
  BatchRows: { request: BatchRowsRequest; data: ImportRowPage }
  DuplicateResolve: { request: DuplicateResolveRequest; data: BatchView }
  BatchCommit: { request: BatchCommitRequest; data: BatchCommitView }
  JobList: { request: JobListRequest; data: JobPage }
  JobDetail: { request: JobDetailRequest; data: JobView }
  JobRetry: { request: JobRetryRequest; data: JobView }
  EmailResend: { request: EmailResendRequest; data: JobView }
  Statistics: { request: StatisticsRequest; data: StatisticsView }
  ExportCreate: { request: ExportCreateRequest; data: ExportView }
  ExportDetail: { request: ExportDetailRequest; data: ExportView }
  ExportDownload: { request: ExportDownloadRequest; data: Empty }
  DeliveryList: { request: DeliveryListRequest; data: DeliveryPage }
  DeliveryRecord: { request: DeliveryRecordRequest; data: DeliveryView }
  CleanupDetail: { request: CleanupDetailRequest; data: CleanupView }
  CleanupRetry: { request: CleanupRetryRequest; data: JobView }
  AuditList: { request: AuditListRequest; data: AuditPage }
  OperationResult: { request: OperationResultRequest; data: OperationView }
  MyGrantList: { request: MyGrantListRequest; data: GrantPage }
  ResourceFields: { request: ResourceFieldsRequest; data: ResourceFieldsView }
  GrantEventOptions: { request: GrantEventOptionsRequest; data: GrantEventOptionsView }
  RegistrationResult: { request: RegistrationResultRequest; data: SubmissionView }
}
