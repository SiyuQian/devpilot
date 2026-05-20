export function greet(name) {
  return "hi " + name;
}

function internalHelper() {
  greet("world");
}

export class Greeter extends Base {
  hello(name) {
    return greet(name);
  }
}

class Base {}
