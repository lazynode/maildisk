pub fn parallel_return<T, T1, T2>(f1: T1, f2: T2) -> Vec<T>
where
    T1: Fn() -> T,
    T2: Fn() -> T,
{
    vec![f1(), f2()]
}
