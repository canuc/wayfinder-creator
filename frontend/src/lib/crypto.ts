function openKeyDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open('openclaw-creator-keys', 1)
    req.onupgradeneeded = () => {
      req.result.createObjectStore('keys')
    }
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

async function generateAndStoreKeypair(): Promise<{ privateKey: CryptoKey; publicKeyPEM: string }> {
  const keyPair = await crypto.subtle.generateKey(
    { name: 'ECDSA', namedCurve: 'P-256' },
    true,
    ['sign', 'verify'],
  )

  const spki = await crypto.subtle.exportKey('spki', keyPair.publicKey)
  const b64 = btoa(String.fromCharCode(...new Uint8Array(spki)))
  const pem =
    '-----BEGIN PUBLIC KEY-----\n' +
    b64.match(/.{1,64}/g)!.join('\n') +
    '\n-----END PUBLIC KEY-----\n'

  const db = await openKeyDB()
  await new Promise<void>((resolve, reject) => {
    const tx = db.transaction('keys', 'readwrite')
    tx.objectStore('keys').put(keyPair.privateKey, 'privateKey')
    tx.objectStore('keys').put(pem, 'publicKeyPEM')
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error)
  })
  db.close()

  return { privateKey: keyPair.privateKey, publicKeyPEM: pem }
}

export async function getOrCreateKeypair(): Promise<{ privateKey: CryptoKey; publicKeyPEM: string }> {
  const db = await openKeyDB()
  const get = (key: string): Promise<unknown> =>
    new Promise((resolve, reject) => {
      const tx = db.transaction('keys', 'readonly')
      const req = tx.objectStore('keys').get(key)
      req.onsuccess = () => resolve(req.result)
      req.onerror = () => reject(req.error)
    })

  const privateKey = (await get('privateKey')) as CryptoKey | undefined
  const publicKeyPEM = (await get('publicKeyPEM')) as string | undefined
  db.close()

  if (privateKey && publicKeyPEM) {
    return { privateKey, publicKeyPEM }
  }
  return generateAndStoreKeypair()
}

export async function hasKeypair(): Promise<boolean> {
  try {
    const db = await openKeyDB()
    const result = await new Promise<unknown>((resolve, reject) => {
      const tx = db.transaction('keys', 'readonly')
      const req = tx.objectStore('keys').get('privateKey')
      req.onsuccess = () => resolve(req.result)
      req.onerror = () => reject(req.error)
    })
    db.close()
    return !!result
  } catch {
    return false
  }
}

function derToRaw(der: Uint8Array): Uint8Array {
  if (der[0] !== 0x30) return der
  let offset = 2
  if (der[1] & 0x80) offset += der[1] & 0x7f

  if (der[offset] !== 0x02) return der
  const rLen = der[offset + 1]
  const rStart = offset + 2
  let rBytes = der.slice(rStart, rStart + rLen)
  offset = rStart + rLen

  if (der[offset] !== 0x02) return der
  const sLen = der[offset + 1]
  const sStart = offset + 2
  let sBytes = der.slice(sStart, sStart + sLen)

  function pad32(bytes: Uint8Array): Uint8Array {
    while (bytes.length > 32 && bytes[0] === 0) bytes = bytes.slice(1)
    const out = new Uint8Array(32)
    out.set(bytes, 32 - bytes.length)
    return out
  }

  const raw = new Uint8Array(64)
  raw.set(pad32(rBytes), 0)
  raw.set(pad32(sBytes), 32)
  return raw
}

export async function signRequest(
  method: string,
  path: string,
  body?: string,
): Promise<Record<string, string>> {
  const { privateKey } = await getOrCreateKeypair()
  const timestamp = Math.floor(Date.now() / 1000).toString()

  let digest = ''
  if (body) {
    const encoded = new TextEncoder().encode(body)
    const hashBuf = await crypto.subtle.digest('SHA-256', encoded)
    digest = btoa(String.fromCharCode(...new Uint8Array(hashBuf)))
  }

  const signingString = method + '\n' + path + '\n' + timestamp + '\n' + digest
  const sigBuf = await crypto.subtle.sign(
    { name: 'ECDSA', hash: 'SHA-256' },
    privateKey,
    new TextEncoder().encode(signingString),
  )

  const rawSig = derToRaw(new Uint8Array(sigBuf))
  const sigB64 = btoa(String.fromCharCode(...rawSig))

  return {
    'X-Signature': sigB64,
    'X-Signature-Timestamp': timestamp,
    'X-Content-Digest': digest,
    'X-Signature-Method': 'ECDSA-P256-SHA256',
  }
}

export async function exportKeypairJSON(): Promise<string> {
  const { privateKey } = await getOrCreateKeypair()
  const jwk = await crypto.subtle.exportKey('jwk', privateKey)
  return JSON.stringify(jwk)
}

export async function importKeypairJSON(jsonStr: string): Promise<{ privateKey: CryptoKey; publicKeyPEM: string }> {
  const jwk = JSON.parse(jsonStr) as JsonWebKey

  const privateKey = await crypto.subtle.importKey(
    'jwk',
    jwk,
    { name: 'ECDSA', namedCurve: 'P-256' },
    true,
    ['sign'],
  )

  const pubJwk = { kty: jwk.kty, crv: jwk.crv, x: jwk.x, y: jwk.y } as JsonWebKey
  const publicKey = await crypto.subtle.importKey(
    'jwk',
    pubJwk,
    { name: 'ECDSA', namedCurve: 'P-256' },
    true,
    ['verify'],
  )
  const spki = await crypto.subtle.exportKey('spki', publicKey)
  const b64 = btoa(String.fromCharCode(...new Uint8Array(spki)))
  const pem =
    '-----BEGIN PUBLIC KEY-----\n' +
    b64.match(/.{1,64}/g)!.join('\n') +
    '\n-----END PUBLIC KEY-----\n'

  const db = await openKeyDB()
  await new Promise<void>((resolve, reject) => {
    const tx = db.transaction('keys', 'readwrite')
    tx.objectStore('keys').put(privateKey, 'privateKey')
    tx.objectStore('keys').put(pem, 'publicKeyPEM')
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error)
  })
  db.close()

  return { privateKey, publicKeyPEM: pem }
}
