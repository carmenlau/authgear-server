import { describe, it, expect } from "@jest/globals";
import {
  escapeMessageFormatText,
  unescapeMessageFormatText,
} from "./messageFormat";

describe("escapeMessageFormatText", () => {
  it("doubles single quotes", () => {
    expect(escapeMessageFormatText("O'Brien")).toEqual("O''Brien");
    expect(escapeMessageFormatText("Sam's App")).toEqual("Sam''s App");
    expect(escapeMessageFormatText("no quotes")).toEqual("no quotes");
    expect(escapeMessageFormatText("''")).toEqual("''''");
  });

  it("wraps curly braces in single-quote pairs", () => {
    expect(escapeMessageFormatText("{App}")).toEqual("'{'App'}'");
    expect(escapeMessageFormatText("{}")).toEqual("'{''}'");
    expect(escapeMessageFormatText("{0}")).toEqual("'{'0'}'");
  });

  it("handles combinations of quotes and braces", () => {
    expect(escapeMessageFormatText("a'{'b")).toEqual("a'''{'''b");
  });
});

describe("unescapeMessageFormatText", () => {
  it("reverses escapeMessageFormatText", () => {
    function test(text: string) {
      expect(unescapeMessageFormatText(escapeMessageFormatText(text))).toEqual(
        text
      );
    }

    test("O'Brien");
    test("Sam's App");
    test("no quotes");
    test("''");
    test("");
    test("{App}");
    test("{}");
    test("{0}");
    test("a'{'b");
    test("~!@#$%^&*()_+=-`[]{}|;':\",./<>?👻");
  });
});
