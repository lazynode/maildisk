mod types {
    pub mod conf;
}
mod lazy;

use imap::Session;
use native_tls::TlsStream;
use std::error::Error;
use std::io::Read;
use std::net::TcpStream;
use std::sync::mpsc::{self, Receiver, SyncSender};
use types::conf;
use sha2::{Sha256, Sha512, Digest};
use base64;

extern crate imap;

fn Put<'a>(
    config: &conf::Type,
    path: &'a Vec<u8>,
    data: &'a Vec<u8>,
) -> Result<Vec<u8>, Box<dyn Error>> {
    let pool = createpool(config)?;
    let mut hash = puts(&pool, config, &TAGDATA, data).clone();
    hash.extend(path);
    let _ = puts(&pool, config, &TAGATTR, &hash);

    Ok(hash)
}
fn Get<'a>(config: &conf::Type, hash: &'a Vec<u8>) -> Result<Vec<u8>, Box<dyn Error>> {
    let pool = createpool(config)?;
    Ok(gets(&pool, config, &TAGDATA, hash))
}
fn Init(config: &conf::Type) -> Result<(), Box<dyn Error>> {
    let domain_port: Vec<_> = config.address.split(':').collect();
    let client = imap::ClientBuilder::new(domain_port[0], domain_port[1].parse()?).native_tls()?;
    let mut session = match client.login(config.username, config.password) {
        Ok(s) => s,
        Err((e, _)) => return Err(Box::new(e)),
    };
    session.create(MAILBOX)?;
    Ok(())
}
fn createpool(
    config: &conf::Type,
) -> Result<
    (
        SyncSender<Option<Session<TlsStream<TcpStream>>>>,
        Receiver<Option<Session<TlsStream<TcpStream>>>>,
    ),
    Box<dyn Error>,
> {
    let (tx, rx): (SyncSender<_>, Receiver<_>) =
        mpsc::sync_channel::<Option<Session<TlsStream<TcpStream>>>>(config.max_conn);
    for _ in 0..config.max_conn {
        tx.send(None)?;
    }
    Ok((tx, rx))
}
fn pickmail(
    pool: &Receiver<Option<Session<TlsStream<TcpStream>>>>,
    config: &conf::Type,
) -> Result<Session<TlsStream<TcpStream>>, Box<dyn Error>> {
    match pool.recv_timeout(std::time::Duration::MAX)? {
        Some(m) => Ok(m),
        None => {
            let domain_port: Vec<_> = config.address.split(':').collect();
            let client =
                imap::ClientBuilder::new(domain_port[0], domain_port[1].parse()?).native_tls()?;
            let mut session = match client.login(config.username, config.password) {
                Ok(s) => s,
                Err((e, _)) => return Err(Box::new(e)),
            };
            session.select(MAILBOX)?;
            Ok(session)
        }
    }
}
fn puts<'a>(
    pool: &(
        SyncSender<Option<Session<TlsStream<TcpStream>>>>,
        Receiver<Option<Session<TlsStream<TcpStream>>>>,
    ),
    config: &conf::Type,
    tag: &'static [u8],
    data: &'a Vec<u8>,
) -> Vec<u8> {
    if data.len() < HARDLIMIT {
        return put(pool, config, tag, data);
    }

    let (sx, sy) = (SOFTLIMIT, (data.len() / SOFTLIMIT + 1) / 2 * SOFTLIMIT);
    let (this, l, r) = (
        data[..sx].to_owned(),
        &data[sx..sy].to_owned(),
        &data[sy..].to_owned(),
    );

    let mut data = lazy::parallel_return(
		|| puts(pool, config, tag, l), 
		|| puts(pool, config, tag, r));
	data.push(this);


    put(pool, config, tag, &data.into_iter().flatten().collect::<Vec<_>>().to_owned())
}

fn gets<'a>(
    pool: &(
        SyncSender<Option<Session<TlsStream<TcpStream>>>>,
        Receiver<Option<Session<TlsStream<TcpStream>>>>,
    ),
    config: &conf::Type,
    tag: &'static [u8],
    hash: &'a Vec<u8>,
) -> Vec<u8> {
    vec![]
}
fn put<'a>(
    pool: &(
        SyncSender<Option<Session<TlsStream<TcpStream>>>>,
        Receiver<Option<Session<TlsStream<TcpStream>>>>,
    ),
    config: &conf::Type,
    tag: &'static [u8],
    data: &'a Vec<u8>,
) -> Vec<u8> {
	assert!(data.len() <= HARDLIMIT, "size error");
	let mut mail = pickmail(&pool.1, config).expect("pickmail failed");
	let mut hash = Sha256::new();
	hash.update(data);
	let hash = hash.finalize();

	let subject = hex::encode(hash);
	let to = hex::encode(tag);
	// let sc = imap::

	let resp = mail.search(format!("Subject {} To {}", subject, to)).unwrap();
	if resp.len() == 0 {
		let content = format!("Subject: {}\r\nTo: {}\r\n\r\n{}", subject, to, base64::encode(data.iter()).to_owned());
		let _ = mail.append(MAILBOX, content.as_bytes()).finish().unwrap();
	}

	pool.0.clone().send(Some(mail)).unwrap();
    hash.to_vec()
}

const MAILBOX: &str = "MDDATA";
const HARDLIMIT: usize = 64 * 1024;
const SOFTLIMIT: usize = HARDLIMIT - 64;
const TAGATTR: &[u8] = "ATTR".as_bytes();
const TAGDATA: &[u8] = "DATA".as_bytes();

#[cfg(test)]
mod tests {
    use super::*;

	const C : conf::Type = conf::Type {
		address: "mail.7.day:993",
		username: "ms01@7.day",
		password: "Ms01M$01",
		max_conn: 4,
	};

    #[test]
    fn init() {
        // todo: delete the "MDDATA" folder before testing
        Init(&C)
        .unwrap();
    }

    #[test]
    fn put_testdata() {
        // todo: delete the "MDDATA" folder before testing
        Put(&C, &"/test".as_bytes().to_vec(),& "test".as_bytes().to_vec()).unwrap();
    }
}
