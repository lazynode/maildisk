pub struct Type<'a> {
    pub address: &'a str,
    pub username: &'a str,
    pub password: &'a str,
    pub max_conn: usize,
}
