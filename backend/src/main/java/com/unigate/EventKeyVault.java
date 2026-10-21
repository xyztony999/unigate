package com.unigate;

import java.nio.charset.StandardCharsets;
import java.nio.file.*;
import java.security.*;
import java.util.*;
import javax.crypto.*;
import javax.crypto.spec.*;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

/**
 * Event data keys live outside database backups. Revocation tombstones must survive database
 * restoration.
 */
@Component
public class EventKeyVault {
  private final Path root;
  private final Crypto master;
  private final SecureRandom random = new SecureRandom();

  public EventKeyVault(
      Crypto master, @Value("${unigate.key-directory:./var/keys}") String directory) {
    this.master = master;
    root = Path.of(directory).toAbsolutePath().normalize();
  }

  private Path file(String event, String suffix) {
    UUID.fromString(event);
    return root.resolve(event + suffix);
  }

  public void requireNotRevoked(String event) {
    if (Files.exists(file(event, ".revoked"))) {
      throw new ApiException(410, "TOKEN_EXPIRED", "活动数据密钥已撤销");
    }
  }

  private synchronized byte[] key(String event, boolean create) throws Exception {
    Files.createDirectories(root);
    if (Files.exists(file(event, ".revoked")))
      throw new ApiException(410, "TOKEN_EXPIRED", "活动数据密钥已撤销");
    Path p = file(event, ".key");
    if (!Files.exists(p)) {
      if (!create) throw new ApiException(410, "TOKEN_EXPIRED", "活动数据密钥不可用");
      byte[] key = new byte[32];
      random.nextBytes(key);
      try {
        Files.write(
            p, master.seal(Base64.getEncoder().encodeToString(key)), StandardOpenOption.CREATE_NEW);
      } catch (FileAlreadyExistsException ignored) {
      }
    }
    return Base64.getDecoder().decode(master.open(Files.readAllBytes(p)));
  }

  public byte[] seal(String event, String text) {
    try {
      byte[] iv = new byte[12];
      random.nextBytes(iv);
      Cipher cipher = Cipher.getInstance("AES/GCM/NoPadding");
      cipher.init(
          Cipher.ENCRYPT_MODE,
          new SecretKeySpec(key(event, true), "AES"),
          new GCMParameterSpec(128, iv));
      cipher.updateAAD(event.getBytes(StandardCharsets.US_ASCII));
      byte[] encrypted = cipher.doFinal(text.getBytes(StandardCharsets.UTF_8));
      byte[] out = new byte[49 + encrypted.length];
      out[0] = 2;
      System.arraycopy(event.getBytes(StandardCharsets.US_ASCII), 0, out, 1, 36);
      System.arraycopy(iv, 0, out, 37, 12);
      System.arraycopy(encrypted, 0, out, 49, encrypted.length);
      return out;
    } catch (ApiException e) {
      throw e;
    } catch (Exception e) {
      throw new IllegalStateException("Event key storage unavailable", e);
    }
  }

  public String open(byte[] data) {
    try {
      String event = new String(data, 1, 36, StandardCharsets.US_ASCII);
      Cipher cipher = Cipher.getInstance("AES/GCM/NoPadding");
      cipher.init(
          Cipher.DECRYPT_MODE,
          new SecretKeySpec(key(event, false), "AES"),
          new GCMParameterSpec(128, Arrays.copyOfRange(data, 37, 49)));
      cipher.updateAAD(event.getBytes(StandardCharsets.US_ASCII));
      return new String(
          cipher.doFinal(Arrays.copyOfRange(data, 49, data.length)), StandardCharsets.UTF_8);
    } catch (ApiException e) {
      throw e;
    } catch (Exception e) {
      throw new ApiException(410, "TOKEN_EXPIRED", "活动数据密钥不可用");
    }
  }

  public synchronized void revoke(String event) {
    try {
      Files.createDirectories(root);
      Files.writeString(
          file(event, ".revoked"),
          java.time.Instant.now().toString(),
          StandardOpenOption.CREATE,
          StandardOpenOption.TRUNCATE_EXISTING);
      Files.deleteIfExists(file(event, ".key"));
    } catch (Exception e) {
      throw new IllegalStateException("Cannot revoke event key", e);
    }
  }
}
