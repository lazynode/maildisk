use std::thread;
use std::sync::mpsc;

type F<'a, T> = Box<dyn Fn() -> T + std::marker::Send + 'a>;

pub fn parallel_return<'a, T>(f1: F<'a, T>, f2: F<'a, T>) -> Vec<T>
where
    T: std::marker::Send + 'a,
{
    let (tx1, rx1) = mpsc::channel();
    let (tx2, rx2) = mpsc::channel();
    let _ = thread::scope::<'a>(move |_| {
        tx1.send(f1()).unwrap();
    });

    let _ = thread::scope::<'a>(move |_| {
        tx2.send(f2()).unwrap();
    });
    vec![rx1.recv().unwrap(), rx2.recv().unwrap()]
}
