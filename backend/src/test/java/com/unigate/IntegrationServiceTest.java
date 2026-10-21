package com.unigate;

import static org.junit.jupiter.api.Assertions.*;

import java.io.*;
import java.nio.charset.StandardCharsets;
import java.util.*;
import org.apache.poi.xssf.usermodel.XSSFWorkbook;
import org.junit.jupiter.api.Test;

class IntegrationServiceTest {
  @Test
  void utf8BomCsvPreservesQuotedNamesAndPhoneText() throws Exception {
    var rows =
        IntegrationService.parseFile(
            "\uFEFF姓名,手机,人数\r\n\"张,三\",+86 138-0000-0000,3\r\n".getBytes(StandardCharsets.UTF_8),
            "名单.csv");
    assertEquals(1, rows.size());
    assertEquals("张,三", rows.getFirst().get("姓名"));
    var fields =
        IntegrationService.normalize(
            rows.getFirst(),
            Store.map(
                "nameColumn",
                "姓名",
                "phoneColumn",
                "手机",
                "countColumn",
                "人数",
                "countMode",
                "TOTAL"));
    assertEquals(2, fields.get("companionCount"));
    assertEquals("+8613800000000", fields.get("phone"));
  }

  @Test
  void noCountColumnMeansNoCompanions() {
    assertEquals(
        0,
        IntegrationService.normalize(Map.of(), Map.of("countMode", "COMPANIONS"))
            .get("companionCount"));
  }

  @Test
  void totalMustIncludeRegistrant() {
    assertThrows(
        ApiException.class,
        () ->
            IntegrationService.normalize(
                Map.of("人数", "0"), Map.of("countColumn", "人数", "countMode", "TOTAL")));
  }

  @Test
  void duplicateHeadersFailInsteadOfOverwriting() {
    assertThrows(
        Exception.class,
        () ->
            IntegrationService.parseFile(
                "手机,手机\n1,2".getBytes(StandardCharsets.UTF_8), "test.csv"));
  }

  @Test
  void xlsxFormulaMarksOnlyItsRowWithoutEvaluation() throws Exception {
    byte[] bytes;
    try (var book = new XSSFWorkbook();
        var out = new ByteArrayOutputStream()) {
      var sheet = book.createSheet();
      sheet.createRow(0).createCell(0).setCellValue("姓名");
      sheet.createRow(1).createCell(0).setCellFormula("1+1");
      book.write(out);
      bytes = out.toByteArray();
    }
    var rows = IntegrationService.parseFile(bytes, "test.xlsx");
    assertEquals(1, rows.size());
    assertTrue(((IntegrationService.ImportRecord) rows.getFirst()).formula);
    assertEquals("", rows.getFirst().get("姓名"));
  }

  @Test
  void xlsxUsesFirstNonemptySheetAndIgnoresOtherSheets() throws Exception {
    byte[] bytes;
    try (var book = new XSSFWorkbook();
        var out = new ByteArrayOutputStream()) {
      book.createSheet("empty");
      var selected = book.createSheet("selected");
      selected.createRow(0).createCell(0).setCellValue("name");
      selected.createRow(1).createCell(0).setCellValue("Parent");
      var ignored = book.createSheet("other");
      ignored.createRow(0).createCell(0).setCellValue("other");
      ignored.createRow(1).createCell(0).setCellFormula("1+1");
      book.write(out);
      bytes = out.toByteArray();
    }
    var rows = IntegrationService.parseFile(bytes, "sheets.xlsx");
    assertEquals(1, rows.size());
    assertEquals("Parent", rows.getFirst().get("name"));
    assertEquals("sheet:1:selected", ((IntegrationService.ImportRecord) rows.getFirst()).sheetRef);
  }

  @Test
  void contactsKeepCountryCodeAndCaseSensitiveEmailLocalPart() {
    assertEquals(
        "+8613800000000", NotificationService.normalizeContact("+86 (138) 0000-0000", "SMS"));
    assertEquals(
        "Parent@example.com", NotificationService.normalizeContact("Parent@EXAMPLE.COM", "EMAIL"));
  }

  @Test
  void emailPrivateLinkUsesActualParticipantRouteAndFragment() {
    var uri =
        java.net.URI.create(
            NotificationService.partyLink("https://unigate.example/", "test-query", "test-change"));
    assertEquals("/me/party", uri.getPath());
    assertNull(uri.getQuery());
    assertEquals("queryToken=test-query&changeToken=test-change", uri.getFragment());
  }

  @Test
  void csvFormulaInjectionIsNeutralized() {
    assertEquals(
        "' =HYPERLINK(\"https://example.invalid\")",
        IntegrationService.safeCell(" =HYPERLINK(\"https://example.invalid\")"));
    assertEquals("'@SUM(A1)", IntegrationService.safeCell("@SUM(A1)"));
    assertEquals("普通文本", IntegrationService.safeCell("普通文本"));
  }
}
