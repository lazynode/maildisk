use std::thread;
use std::sync::mpsc;

pub fn parallel_return<'a, T>(f1: Box<dyn Fn() -> T + std::marker::Send + 'a>, f2: Box<dyn Fn() -> T + std::marker::Send + 'a>) -> Vec<T>
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
