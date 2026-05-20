export function greet(name: string): string {
  return "hi " + name;
}

function internalHelper(): void {
  greet("world");
}

export class Greeter extends Base implements Speaker {
  hello(name: string): string {
    return greet(name);
  }
  private silent(): void {}
}

class Base {}

export interface Speaker {
  hello(name: string): string;
}

export type Greeting = string;
