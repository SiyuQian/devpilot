export function greet(name: string): string {
  return "hi " + name;
}

function internalHelper(): void {
  greet("world");
}

export class Greeter {
  hello(name: string): string {
    return greet(name);
  }
  private silent(): void {}
}

export interface Speaker {
  hello(name: string): string;
}

export type Greeting = string;
