import { parseArgs } from 'util'
import { RPC, encodeDirectory } from 'soroban-client-nodejs'
import { User } from './user.js'

async function demo(directoryName, hashEncode, rpc) {
  // Alice
  console.log('')
  console.log('=== Alice ===')
  // Generate private key
  const alice = await User.createNewUser()
  console.log(`Alice: pubKey = ${alice.publicKey()}`)

  console.log('Alice: Add candidate in directory')
  let payload = User.nacl.to_hex(alice.publicKey())
  await rpc.directoryAdd(directoryName, payload, 'long')

  // Bob
  console.log('')
  console.log('=== Bob ===')
  // Generate private key
  const bob = await User.createNewUser()
  console.log(`Bob: pubKey = ${bob.publicKey()}`)

  // Wait for candidate
  const alicePublicKeyStr = await rpc.waitAndRemove(directoryName)
  const alicePublicKey = User.nacl.from_hex(alicePublicKeyStr)
  
  console.log('Bob: Send pubkey to Alice private directory')
  let privateDirectory = directoryName + '.' + alicePublicKeyStr
  if (hashEncode)
    privateDirectory = encodeDirectory(privateDirectory)

  payload = User.nacl.to_hex(bob.publicKey())
  await rpc.directoryAdd(privateDirectory, payload, 'default')

  // Get shared directory    
  let sharedDirectory = encodeDirectory(bob.boxShared(alicePublicKey))
  console.log(`Bob: Shared directory ${sharedDirectory}`)

  // Alice
  console.log('')
  console.log('=== Alice ===')

  // Wait for candidate
  const bobPublicKeyStr = await rpc.waitAndRemove(privateDirectory)
  const bobPublicKey = User.nacl.from_hex(bobPublicKeyStr)

  // Remove offer
  console.log('Alice: Remove candidate from directory')
  payload = User.nacl.to_hex(alice.publicKey())
  await rpc.directoryRemove(directoryName, payload)

  // Get shared directory    
  sharedDirectory = encodeDirectory(alice.boxShared(bobPublicKey))
  console.log(`Alice: Shared directory = ${sharedDirectory}`)

  payload = alice.boxEncrypt('Hello From Alice', bobPublicKey)
  if (payload == null)
    return

  await rpc.directoryAdd(sharedDirectory, payload, 'short')

  const aliceNextDirectory = encodeDirectory(payload)
  console.log(`Alice: Next directory = ${aliceNextDirectory}`)

  console.log('')
  console.log('=== Bob ===')
  // Consider last response
  payload = await rpc.waitAndRemove(sharedDirectory)

  let message = bob.boxDecrypt(payload, alicePublicKey)
  if (message == null)
    return
  const responseDirectory = encodeDirectory(payload)
  console.log(`Bob: Alice message = ${message}`)

  payload = bob.boxEncrypt('Hi from Bob', alicePublicKey)
  if (payload == null)
    return

  await rpc.directoryAdd(responseDirectory, payload, 'short')

  console.log('')
  console.log('=== Alice ===')
  // Consider last response
  payload = await rpc.waitAndRemove(aliceNextDirectory)

  message = alice.boxDecrypt(payload, bobPublicKey)
  if (message == null)
    return

  console.log(`Alice: Bob Response = ${message}`)

}

// Initialize demo parameters
let sorobanUrl = 'http://sorg3sf2lxhd6swneuuzvo7jluuassw5qsxakzgimr5agvkj35265gad.onion/rpc'
let proxyUrl = 'http://127.0.0.1:9150'

// Parse command line arguments
const {values, _} = parseArgs({
  'args': process.argv.slice(2),
  'options': {
    'soroban': {
      type: 'string',
      short: 's',
    },
    'proxy': {
      type: 'string',
      short: 'p',
    },
  }
})

if ('soroban' in values) {
  sorobanUrl = values['soroban']
}

if ('proxy' in values) {
  proxyUrl = values['proxy']
}

// Initialize the Soroban RPC client 
const client = new RPC(sorobanUrl, proxyUrl)

// Launch the demo
await demo('soroban.demo', true, client)
