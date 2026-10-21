package com.unigate;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.Base64;
import org.springframework.security.crypto.bcrypt.BCryptPasswordEncoder;
import org.springframework.security.crypto.password.PasswordEncoder;

/**
 * Versioned, domain separated SHA-256 prehash keeps the full password within bcrypt's byte limit.
 */
public final class PasswordHasher implements PasswordEncoder {
  private static final String PREFIX = "$ug$sha256-bcrypt$v1$";
  private final BCryptPasswordEncoder bcrypt = new BCryptPasswordEncoder(12);

  private String prehash(CharSequence password) {
    try {
      MessageDigest digest = MessageDigest.getInstance("SHA-256");
      digest.update("UniGate password v1\0".getBytes(StandardCharsets.UTF_8));
      return Base64.getEncoder()
          .encodeToString(digest.digest(password.toString().getBytes(StandardCharsets.UTF_8)));
    } catch (java.security.NoSuchAlgorithmException error) {
      throw new IllegalStateException(error);
    }
  }

  @Override
  public String encode(CharSequence password) {
    return PREFIX + bcrypt.encode(prehash(password));
  }

  @Override
  public boolean matches(CharSequence password, String stored) {
    if (password == null || stored == null) return false;
    if (stored.startsWith(PREFIX))
      return bcrypt.matches(prehash(password), stored.substring(PREFIX.length()));
    // Existing installations retain their legacy bcrypt hashes; no password or account reset.
    if (password.toString().getBytes(StandardCharsets.UTF_8).length > 72) return false;
    return bcrypt.matches(password, stored);
  }

  @Override
  public boolean upgradeEncoding(String stored) {
    return stored != null && !stored.startsWith(PREFIX);
  }
}
