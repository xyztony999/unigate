package com.unigate;

import static org.junit.jupiter.api.Assertions.*;

import com.fasterxml.jackson.databind.ObjectMapper;
import java.util.*;
import org.junit.jupiter.api.Test;

class ContractTest {
  final Contract contract = new Contract(new ObjectMapper());

  @Test
  void rejectsUnknownFieldsWrongTypesAndMalformedOperationIds() {
    assertThrows(
        ApiException.class,
        () -> contract.validate("GateCommitRequest", Map.of("operationId", "bad")));
    assertThrows(
        ApiException.class,
        () -> contract.validate("SessionBootstrapRequest", Map.of("secret", "x")));
    assertThrows(
        ApiException.class,
        () ->
            contract.validate(
                "SubmitRegistrationRequest",
                Map.of(
                    "operationId",
                    UUID.randomUUID().toString(),
                    "eventId",
                    UUID.randomUUID().toString(),
                    "fields",
                    Map.of("companionCount", -1),
                    "acceptedNoticeVersion",
                    "v1")));
  }

  @Test
  void allOperationsRetainOriginalContractAndDistinctRoutes() {
    assertEquals(85, contract.operations.size());
    assertEquals(85, contract.routes.size());
    for (var op : contract.operations.values()) {
      assertTrue(Contract.schemas.containsKey(op.get("request")));
      assertTrue(Contract.schemas.containsKey(op.get("data")));
    }
  }

  @Test
  void capabilitiesUseAuthenticatedEncryptionAndRejectTampering() {
    Crypto c = new Crypto(Base64.getEncoder().encodeToString(new byte[32]));
    String token = c.token();
    assertEquals(43, token.length());
    byte[] encrypted = c.seal("private");
    assertFalse(Arrays.equals("private".getBytes(), encrypted));
    assertEquals("private", c.open(encrypted));
    encrypted[encrypted.length - 1] ^= 1;
    assertThrows(ApiException.class, () -> c.open(encrypted));
  }

  @Test
  void revokedEventKeyCannotDecryptOldDatabaseCiphertextOrBeRecreated(
      @org.junit.jupiter.api.io.TempDir java.nio.file.Path temp) {
    Crypto c = new Crypto(Base64.getEncoder().encodeToString(new byte[32]));
    EventKeyVault vault = new EventKeyVault(c, temp.toString());
    String event = UUID.randomUUID().toString();
    byte[] oldBackupCipher = vault.seal(event, "participant detail");
    assertEquals("participant detail", vault.open(oldBackupCipher));
    vault.revoke(event);
    assertThrows(ApiException.class, () -> vault.open(oldBackupCipher));
    assertThrows(ApiException.class, () -> vault.seal(event, "must not resurrect"));
    assertFalse(java.nio.file.Files.exists(temp.resolve(event + ".key")));
    assertTrue(java.nio.file.Files.exists(temp.resolve(event + ".revoked")));
  }
}
