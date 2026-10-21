package com.unigate;

import java.nio.charset.StandardCharsets;
import java.security.*;
import java.util.*;
import javax.crypto.*;
import javax.crypto.spec.*;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

@Component
public class Crypto {
  private final byte[] key;
  private final SecureRandom random = new SecureRandom();

  public Crypto(@Value("${unigate.master-key}") String encoded) {
    try {
      key = Base64.getDecoder().decode(encoded);
      if (key.length != 32) throw new IllegalArgumentException();
    } catch (Exception e) {
      throw new IllegalStateException(
          "UNIGATE_MASTER_KEY must be a Base64 encoded 32 byte deployment secret");
    }
  }

  public String token() {
    byte[] b = new byte[32];
    random.nextBytes(b);
    return Base64.getUrlEncoder().withoutPadding().encodeToString(b);
  }

  public byte[] hash(String s) {
    try {
      Mac m = Mac.getInstance("HmacSHA256");
      m.init(new SecretKeySpec(key, "HmacSHA256"));
      return m.doFinal(s.getBytes(StandardCharsets.UTF_8));
    } catch (Exception e) {
      throw new IllegalStateException(e);
    }
  }

  public byte[] seal(String text) {
    try {
      byte[] iv = new byte[12];
      random.nextBytes(iv);
      Cipher c = Cipher.getInstance("AES/GCM/NoPadding");
      c.init(Cipher.ENCRYPT_MODE, new SecretKeySpec(key, "AES"), new GCMParameterSpec(128, iv));
      byte[] crypt = c.doFinal(text.getBytes(StandardCharsets.UTF_8));
      byte[] out = new byte[13 + crypt.length];
      out[0] = 1;
      System.arraycopy(iv, 0, out, 1, 12);
      System.arraycopy(crypt, 0, out, 13, crypt.length);
      return out;
    } catch (Exception e) {
      throw new IllegalStateException(e);
    }
  }

  public String open(byte[] b) {
    try {
      if (b[0] != 1) throw new IllegalArgumentException();
      Cipher c = Cipher.getInstance("AES/GCM/NoPadding");
      c.init(
          Cipher.DECRYPT_MODE,
          new SecretKeySpec(key, "AES"),
          new GCMParameterSpec(128, Arrays.copyOfRange(b, 1, 13)));
      return new String(c.doFinal(Arrays.copyOfRange(b, 13, b.length)), StandardCharsets.UTF_8);
    } catch (Exception e) {
      throw new ApiException(410, "CAPABILITY_EXPIRED", "凭证已失效");
    }
  }

  public String signed(Map<String, Object> data, com.fasterxml.jackson.databind.ObjectMapper json) {
    try {
      return Base64.getUrlEncoder()
          .withoutPadding()
          .encodeToString(seal(json.writeValueAsString(data)));
    } catch (Exception e) {
      throw new IllegalStateException(e);
    }
  }

  public Map<String, Object> verify(String s, com.fasterxml.jackson.databind.ObjectMapper json) {
    try {
      return json.readValue(open(Base64.getUrlDecoder().decode(s)), Map.class);
    } catch (Exception e) {
      throw new ApiException(410, "CAPABILITY_EXPIRED", "凭证已失效");
    }
  }
}
