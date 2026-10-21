package com.unigate;

import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

@Component
public class LifecycleWorker {
  private final Store s;
  private final IntegrationService integration;

  public LifecycleWorker(Store s, IntegrationService integration) {
    this.s = s;
    this.integration = integration;
  }

  @Scheduled(fixedDelayString = "${unigate.lifecycle.poll-ms:60000}")
  @Transactional
  public void expireNext() {
    try {
      var due =
          s.rows(
              "select c.* from ug_cleanup c join ug_event e on c.event_id=e.id where"
                  + " c.due_at<=now() and c.status='PENDING' order by c.due_at,c.id for update of e"
                  + " skip locked limit 1");
      if (!due.isEmpty()) {
        var row = s.row("select * from ug_cleanup where id=? for update", due.getFirst().get("id"));
        integration.runCleanup(row);
      }
    } finally {
      s.eventScope.remove();
    }
  }
}
