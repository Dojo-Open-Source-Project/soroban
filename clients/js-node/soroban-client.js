import { parseArgs } from 'util'
import { RPC, encodeDirectory } from 'soroban-client-nodejs'
import { User } from './user.js'


async function initiator(url, directoryName, hashEncode, rpc, numIter) {
  const user = await User.createNewUser()

  console.log("Registering public key")
  let payload = User.nacl.to_hex(user.publicKey())
  await rpc.directoryAdd(directoryName, payload, 'long')
  
  let privateDirectory = `${directoryName}.${payload}`
  if (hashEncode)
    privateDirectory = encodeDirectory(privateDirectory)
  if (privateDirectory == null) {
    console.log('Invalid private directory')
    return
  }

  const candidatePublicKeyStr = await rpc.waitAndRemove(privateDirectory, 100)
  if (candidatePublicKeyStr == null) {
    console.log('Invalid candidate public key')
    return
  }
  console.log('Contributor public key found')
  const candidatePublicKey = User.nacl.from_hex(candidatePublicKeyStr)

  let nextDirectory = encodeDirectory(user.boxShared(candidatePublicKey))
  if (nextDirectory == null)
    return

  payload = ''

  console.log('Starting exchange loop...')
  let counter = 1

  while (numIter > 0) {
    // request
    console.log('Sending : Ping')
    let msg = `Ping ${counter} ${Date.now()}`
    payload = user.boxEncrypt(msg, candidatePublicKey)
    if (payload == null) {
      console.log('Invalid Ping message')
      return
    }
    await rpc.directoryAdd(nextDirectory, payload, 'short')
    nextDirectory = encodeDirectory(payload)
    if (nextDirectory == null) {
      console.log('Invalid response next directory')
      return
    }

    // response
    payload = await rpc.waitAndRemove(nextDirectory, 10)
    nextDirectory = encodeDirectory(payload)
    if (nextDirectory == null) {
      console.log('Invalid response next directory')
      return
    }

    msg = user.boxDecrypt(payload, candidatePublicKey)
    if (msg == null) {
      console.log('Invalid reponse message')
      return
    }
    console.log(`Received: ${msg}`)
    counter += 1
    numIter -= 1
  }
}


async function contributor(url, directoryName, hashEncode, rpc, numIter) {
  const user = await User.createNewUser()

  const initiatorPublicKeyStr = await rpc.waitAndRemove(directoryName, 10)
  if (initiatorPublicKeyStr == null) {
    console.log('Invalid initiator public key')
    return
  }

  const initiatorPublicKey = User.nacl.from_hex(initiatorPublicKeyStr)

  console.log('Initiator public key found')

  let privateDirectory = `${directoryName}.${initiatorPublicKeyStr}`
  if (hashEncode)
    privateDirectory = encodeDirectory(privateDirectory)

  if (privateDirectory == null) {
    console.log("Invalid private directory")
    return
  }

  console.log("Sending public key")
  let msg = User.nacl.to_hex(user.publicKey())
  await rpc.directoryAdd(privateDirectory, msg, 'default')

  let nextDirectory = encodeDirectory(user.boxShared(initiatorPublicKey))
  if (nextDirectory == null) {
    console.log('Invalid next directory (start)')
    return
  }

  let payload = ''

  console.log('Starting exchange loop...')
  let counter = 1

  while (numIter > 0) {
    // Query
    payload = await rpc.waitAndRemove(nextDirectory, 10)
    if (payload == null) {
      console.log('Invalid payload (query)')
      return
    }
    nextDirectory = encodeDirectory(payload)
    if (nextDirectory == null) {
      console.log("Invalid next_directory (query)", next_directory)
      return
    }

    const message = user.boxDecrypt(payload, initiatorPublicKey)
    if (message == null) {
      console.log('Invalid query')
      return
    }

    console.log(`Received: ${message}`)

    // Response
    console.log('Replying: Pong')
    let msg = `Pong ${counter} ${Date.now()}`
    payload = user.boxEncrypt(msg, initiatorPublicKey)
    if (payload == null) {
      console.log('Invalid payload (reply)')
      return
    }
    await rpc.directoryAdd(nextDirectory, payload, 'short')

    nextDirectory = encodeDirectory(payload)
    if (nextDirectory == null) {
      console.log('Invalid next directory (response)')
      return
    }

    counter += 1
    numIter -= 1
  }
}

async function main(argv) {
  // Parse command line arguments
  const {values, _} = parseArgs({
    'args': argv,
    'options': {
      'url': {
        type: 'string',
        short: 'u',
      },
      'with_tor': {
        type: 'string',
        short: 't',
      },
      'directory_name': {
        type: 'string',
        short: 'd',
      },
      'role': {
        type: 'string',
        short: 'r',
      },
      'num_iter': {
        type: 'string',
        short: 'n',
      },
      'hash_encode': {
        type: 'boolean',
        short: 'e',
      },
    }
  })

  let url = ('url' in values) ? values['url'] : ''
  const withTor = ('with_tor' in values) ? values['with_tor'] : ''
  let directoryName = ('directory_name' in values) ? values['directory_name'] : 'test'
  const role = ('role' in values) ? values['role'] : 'demo'
  const numIter = ('num_iter' in values) ? parseInt(values['num_iter']) : 3
  const hashEncode = ('hash_encode' in values) ? values['hash_encode'] : false

  if (hashEncode)
    directoryName = encodeDirectory(directoryName)

  if (directoryName == null)
    return

  const client = new RPC(url, withTor)

  try {
    while (true) {
      if (role == 'initiator') {
        await initiator(url, directoryName, hashEncode, client, numIter)
      } else if (role == 'contributor') {
        await contributor(url, directoryName, hashEncode, client, numIter)
        await new Promise(r => setTimeout(r, 2000));
      } else {
        throw Error('Invalid role')
      }
      console.log('Done')
    }
  } catch (e) {
    console.log(`Error: ${e}`)
  }
}

// Start script
const argv = process.argv.slice(2)
await main(argv)
