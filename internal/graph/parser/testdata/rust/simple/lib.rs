pub fn greet(name: &str) -> String {
    format!("hi {}", name)
}

fn internal_helper() {
    greet("world");
}

pub struct Greeter {
    pub name: String,
}

pub enum Mood { Happy, Sad }

pub type Greeting = String;
