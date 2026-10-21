package com.unigate;

import static com.unigate.Store.*;

import java.util.*;
import org.springframework.stereotype.Service;
import org.springframework.transaction.*;
import org.springframework.transaction.support.TransactionTemplate;

/** Failed authentication counters commit independently of the rejected business transaction. */
@Service
public class SecurityAttempts {
  final Store s;
  final TransactionTemplate tx;

  public SecurityAttempts(Store s, PlatformTransactionManager tm) {
    this.s = s;
    tx = new TransactionTemplate(tm);
    tx.setPropagationBehavior(TransactionDefinition.PROPAGATION_REQUIRES_NEW);
  }

  public void checkLogin(String username, String ip) {
    String scope = scope(username, ip);
    int count =
        s.db.queryForObject(
            "select count(*) from ug_challenge where kind='LOGIN_LIMIT' and actor_scope=? and"
                + " expires_at>now()",
            Integer.class,
            scope);
    ApiException.require(count < 5, 429, "RATE_LIMITED", "登录尝试过多，请稍后再试");
  }

  String scope(String username, String ip) {
    return "LOGIN:" + HexFormat.of().formatHex(s.crypto.hash(username + ":" + ip));
  }

  public void failedLogin(String username, String ip) {
    tx.executeWithoutResult(
        st ->
            s.execute(
                "insert into ug_challenge(id,kind,actor_scope,code_digest,expires_at)"
                    + " values(?,'LOGIN_LIMIT',?,?,now()+interval '15 minutes')",
                uuid(s.id()),
                scope(username, ip),
                s.crypto.hash(s.crypto.token())));
  }

  public void failedPairing(Object id) {
    tx.executeWithoutResult(
        st ->
            s.execute(
                "update ug_device set pairing_failures=least(pairing_failures+1,5) where id=?",
                id));
  }
}
