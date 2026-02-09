// SSH ed25519 keypair generation using Web Crypto API

function encodeUint32BE(value: number): Uint8Array {
  const buf = new Uint8Array(4)
  buf[0] = (value >>> 24) & 0xff
  buf[1] = (value >>> 16) & 0xff
  buf[2] = (value >>> 8) & 0xff
  buf[3] = value & 0xff
  return buf
}

function encodeSSHString(data: Uint8Array | string): Uint8Array {
  const bytes = typeof data === 'string' ? new TextEncoder().encode(data) : data
  const length = encodeUint32BE(bytes.length)
  const result = new Uint8Array(4 + bytes.length)
  result.set(length)
  result.set(bytes, 4)
  return result
}

function concat(...arrays: Uint8Array[]): Uint8Array {
  const total = arrays.reduce((sum, a) => sum + a.length, 0)
  const result = new Uint8Array(total)
  let offset = 0
  for (const a of arrays) {
    result.set(a, offset)
    offset += a.length
  }
  return result
}

function base64Encode(data: Uint8Array): string {
  return btoa(String.fromCharCode(...data))
}

function wrapBase64(b64: string, lineWidth: number): string {
  const lines: string[] = []
  for (let i = 0; i < b64.length; i += lineWidth) {
    lines.push(b64.slice(i, i + lineWidth))
  }
  return lines.join('\n')
}

export function downloadFile(filename: string, content: string) {
  const blob = new Blob([content], { type: 'application/octet-stream' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

export async function isEd25519Supported(): Promise<boolean> {
  try {
    await crypto.subtle.generateKey('Ed25519', true, ['sign', 'verify'])
    return true
  } catch {
    return false
  }
}

export async function generateSSHKeypair(): Promise<{
  privateKeyPEM: string
  publicKeySSH: string
}> {
  const keyPair = await crypto.subtle.generateKey(
    'Ed25519' as unknown as EcKeyGenParams,
    true,
    ['sign', 'verify'],
  )

  // Export raw public key (32 bytes)
  const rawPub = new Uint8Array(await crypto.subtle.exportKey('raw', keyPair.publicKey))

  // Export PKCS8 private key to extract raw private bytes
  const pkcs8 = new Uint8Array(await crypto.subtle.exportKey('pkcs8', keyPair.privateKey))
  // PKCS8 for ed25519: last 32 bytes of the DER are the private key seed,
  // wrapped in an OCTET STRING (04 20 <32 bytes>) at the end
  const rawPriv = pkcs8.slice(pkcs8.length - 32)

  // Build SSH public key: "ssh-ed25519 <base64>"
  const keyType = 'ssh-ed25519'
  const pubBlob = concat(encodeSSHString(keyType), encodeSSHString(rawPub))
  const publicKeySSH = `${keyType} ${base64Encode(pubBlob)}`

  // Build OpenSSH private key format
  const AUTH_MAGIC = new TextEncoder().encode('openssh-key-v1\0')
  const ciphername = encodeSSHString('none')
  const kdfname = encodeSSHString('none')
  const kdf = encodeSSHString(new Uint8Array(0))
  const nkeys = encodeUint32BE(1)
  const publicKeySection = encodeSSHString(pubBlob)

  // Private section (unencrypted)
  const checkBytes = crypto.getRandomValues(new Uint8Array(4))
  const checkInt = concat(checkBytes, checkBytes) // same 4 bytes repeated

  const privPayload = concat(
    checkInt,
    encodeSSHString(keyType),       // key type
    encodeSSHString(rawPub),         // public key
    encodeSSHString(concat(rawPriv, rawPub)), // private key (64 bytes: seed + pub)
    encodeSSHString(''),             // comment
  )

  // Pad to block size (8 for "none" cipher)
  const blockSize = 8
  const padLen = blockSize - (privPayload.length % blockSize)
  const padding = new Uint8Array(padLen)
  for (let i = 0; i < padLen; i++) padding[i] = i + 1
  const paddedPriv = concat(privPayload, padding)
  const privateSection = encodeSSHString(paddedPriv)

  const fullBlob = concat(AUTH_MAGIC, ciphername, kdfname, kdf, nkeys, publicKeySection, privateSection)
  const b64Body = wrapBase64(base64Encode(fullBlob), 70)

  const privateKeyPEM = `-----BEGIN OPENSSH PRIVATE KEY-----\n${b64Body}\n-----END OPENSSH PRIVATE KEY-----\n`

  return { privateKeyPEM, publicKeySSH }
}
