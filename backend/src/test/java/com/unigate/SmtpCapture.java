package com.unigate;

import java.io.*;
import java.net.*;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.*;

/** Test-only loopback SMTP sink. It has no forwarding or external network capability. */
final class SmtpCapture implements AutoCloseable {
  private final ServerSocket server;
  private final CompletableFuture<String> body = new CompletableFuture<>();
  private final Thread thread;

  SmtpCapture(Runnable onData) throws IOException {
    server = new ServerSocket(0, 1, InetAddress.getByName("127.0.0.1"));
    thread =
        Thread.ofPlatform()
            .daemon()
            .start(
                () -> {
                  try (var socket = server.accept();
                      var reader =
                          new BufferedReader(
                              new InputStreamReader(
                                  socket.getInputStream(), StandardCharsets.UTF_8));
                      var writer =
                          new PrintWriter(
                              new OutputStreamWriter(
                                  socket.getOutputStream(), StandardCharsets.UTF_8),
                              true)) {
                    writer.print("220 localhost test SMTP\r\n");
                    writer.flush();
                    String line;
                    while ((line = reader.readLine()) != null) {
                      if (line.startsWith("EHLO") || line.startsWith("HELO")) {
                        writer.print("250-localhost\r\n250 8BITMIME\r\n");
                      } else if (line.equals("DATA")) {
                        writer.print("354 Send message\r\n");
                        writer.flush();
                        StringBuilder data = new StringBuilder();
                        while ((line = reader.readLine()) != null && !line.equals("."))
                          data.append(line).append("\r\n");
                        onData.run();
                        body.complete(data.toString());
                        writer.print("250 Accepted\r\n");
                      } else if (line.equals("QUIT")) {
                        writer.print("221 Bye\r\n");
                        writer.flush();
                        break;
                      } else writer.print("250 OK\r\n");
                      writer.flush();
                    }
                  } catch (Throwable error) {
                    body.completeExceptionally(error);
                  }
                });
  }

  int port() {
    return server.getLocalPort();
  }

  String message() throws Exception {
    return body.get(10, TimeUnit.SECONDS);
  }

  public void close() throws IOException {
    server.close();
    thread.interrupt();
  }
}
