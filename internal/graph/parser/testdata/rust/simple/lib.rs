use std::fmt::Display;
use crate::other::helper;

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

pub trait Hello {
    fn hello(&self) -> String;
}

impl Hello for Greeter {
    fn hello(&self) -> String {
        greet(&self.name)
    }
}
