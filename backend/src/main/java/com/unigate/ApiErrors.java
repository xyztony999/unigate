package com.unigate;

import static com.unigate.Store.map;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.multipart.MaxUploadSizeExceededException;

@RestControllerAdvice
public class ApiErrors {
  @ExceptionHandler(MaxUploadSizeExceededException.class)
  ResponseEntity<?> oversizedUpload(MaxUploadSizeExceededException error) {
    return ResponseEntity.unprocessableEntity()
        .header("Cache-Control", "no-store")
        .body(
            map(
                "code",
                "FIELD_VALIDATION_FAILED",
                "message",
                "文件不得超过 10MiB",
                "data",
                map(),
                "requestId",
                java.util.UUID.randomUUID().toString()));
  }
}
