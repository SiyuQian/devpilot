import { greet } from "./main";

describe("greet", () => {
  it("greets", () => {
    greet("world");
  });
  test("greets again", () => {
    greet("again");
  });
});
