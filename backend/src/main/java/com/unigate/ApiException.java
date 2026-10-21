package com.unigate;

public class ApiException extends RuntimeException {
  public final int status;
  public final String code;

  public ApiException(int status, String code, String message) {
    super(message);
    this.code = canonical(status, code);
    this.status = this.code.equals("IDEMPOTENCY_PAYLOAD_MISMATCH") ? 400 : status;
  }

  private static String canonical(int status, String code) {
    return switch (code) {
      case "IDEMPOTENCY_CONFLICT" -> "IDEMPOTENCY_PAYLOAD_MISMATCH";
      case "CAPACITY_CONFLICT" -> "CAPACITY_EXCEEDED";
      case "ADMISSION_BALANCE_CONFLICT" -> "PARTY_REMAINING_EXCEEDED";
      case "CSRF_REJECTED", "ORIGIN_REJECTED" -> "CSRF_INVALID";
      case "AUTHENTICATION_FAILED", "DEVICE_UNAUTHORIZED", "PAIRING_INVALID" -> "UNAUTHENTICATED";
      case "VALIDATION_FAILED",
          "FIELD_DISABLED",
          "INVALID_FILE",
          "INVALID_ENCODING",
          "INVALID_PROVIDER" ->
          "FIELD_VALIDATION_FAILED";
      case "PHONE_VERIFICATION_REQUIRED" -> "VERIFICATION_REQUIRED";
      case "PAIRING_EXPIRED",
          "CAPABILITY_EXPIRED",
          "SCAN_EXPIRED",
          "DATA_EXPIRED",
          "RESULT_EXPIRED",
          "BATCH_EXPIRED" ->
          "TOKEN_EXPIRED";
      case "REGISTRATION_CLOSED",
          "EVENT_PHASE_CONFLICT",
          "PARTY_STATE_CONFLICT",
          "REVIEW_STATE_CONFLICT",
          "LAST_ADMIN",
          "LAST_ORGANIZER",
          "CORRECTION_CONFLICT",
          "ADMISSION_HISTORY_EXISTS",
          "POLICY_IN_USE",
          "NOTICE_CHANGED",
          "CHECKIN_CLOSED",
          "DATA_CONFLICT" ->
          "EVENT_STATE_CONFLICT";
      case "PASSWORD_CHANGE_REQUIRED", "SELF_SERVICE_DISABLED" -> "FORBIDDEN";
      default -> {
        if (java.util.Set.of(
                "INVALID_REQUEST",
                "IDEMPOTENCY_PAYLOAD_MISMATCH",
                "UNAUTHENTICATED",
                "SESSION_EXPIRED",
                "FORBIDDEN",
                "CSRF_INVALID",
                "RESOURCE_NOT_FOUND",
                "OPERATION_NOT_FOUND",
                "VERSION_CONFLICT",
                "CAPACITY_EXCEEDED",
                "PARTY_REMAINING_EXCEEDED",
                "EVENT_STATE_CONFLICT",
                "PUBLIC_MUTATIONS_FROZEN",
                "PREVIEW_STALE",
                "SOURCE_CONFLICT",
                "OPERATION_IN_PROGRESS",
                "TOKEN_EXPIRED",
                "ARTIFACT_EXPIRED",
                "FIELD_VALIDATION_FAILED",
                "TICKET_UNAVAILABLE",
                "CREDENTIAL_INVALID",
                "VERIFICATION_REQUIRED",
                "DUPLICATE_REVIEW_REQUIRED",
                "RATE_LIMITED",
                "INTERNAL_ERROR",
                "SERVICE_UNAVAILABLE",
                "PROVIDER_UNAVAILABLE")
            .contains(code)) yield code;
        yield switch (status) {
          case 400 -> "INVALID_REQUEST";
          case 401 -> "UNAUTHENTICATED";
          case 403 -> "FORBIDDEN";
          case 404 -> "RESOURCE_NOT_FOUND";
          case 409 -> "EVENT_STATE_CONFLICT";
          case 410 -> "ARTIFACT_EXPIRED";
          case 422 -> "FIELD_VALIDATION_FAILED";
          case 429 -> "RATE_LIMITED";
          case 503 -> "SERVICE_UNAVAILABLE";
          default -> "INTERNAL_ERROR";
        };
      }
    };
  }

  public static void require(boolean b, int status, String code, String message) {
    if (!b) throw new ApiException(status, code, message);
  }
}
