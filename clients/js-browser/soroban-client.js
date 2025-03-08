
function log(msg) {
  const div = document.getElementById('logs')
  div.innerHTML += `<p>${msg}</p>`
}

async function initiator(url, directoryName, hashEncode, rpc, numIter) {
  const user = await User.createNewUser()

  log("Registering public key")
  let payload = User.nacl.to_hex(user.publicKey())
  await rpc.directoryAdd(directoryName, payload, 'long')
  
  let privateDirectory = `${directoryName}.${payload}`
  if (hashEncode)
    privateDirectory = await encodeDirectory(privateDirectory)
  if (privateDirectory == null) {
    log('Invalid private directory')
    return
  }

  const candidatePublicKeyStr = await rpc.waitAndRemove(privateDirectory, 100)
  if (candidatePublicKeyStr == null) {
    log('Invalid candidate public key')
    return
  }
  log('Contributor public key found')
  const candidatePublicKey = User.nacl.from_hex(candidatePublicKeyStr)

  let nextDirectory = await encodeDirectory(user.boxShared(candidatePublicKey))
  if (nextDirectory == null)
    return

  payload = ''

  log('Starting exchange loop...')
  let counter = 1

  while (numIter > 0) {
    // request
    log('Sending : Ping')
    let msg = `Ping ${counter} ${Date.now()}`
    payload = user.boxEncrypt(msg, candidatePublicKey)
    if (payload == null) {
      log('Invalid Ping message')
      return
    }
    await rpc.directoryAdd(nextDirectory, payload, 'short')
    nextDirectory = await encodeDirectory(payload)
    if (nextDirectory == null) {
      log('Invalid response next directory')
      return
    }

    // response
    payload = await rpc.waitAndRemove(nextDirectory, 10)
    nextDirectory = await encodeDirectory(payload)
    if (nextDirectory == null) {
      log('Invalid response next directory')
      return
    }

    msg = user.boxDecrypt(payload, candidatePublicKey)
    if (msg == null) {
      log('Invalid reponse message')
      return
    }
    log(`Received: ${msg}`)
    counter += 1
    numIter -= 1
  }
}


async function contributor(url, directoryName, hashEncode, rpc, numIter) {
  const user = await User.createNewUser()

  const initiatorPublicKeyStr = await rpc.waitAndRemove(directoryName, 10)
  if (initiatorPublicKeyStr == null) {
    log('Invalid initiator public key')
    return
  }

  const initiatorPublicKey = User.nacl.from_hex(initiatorPublicKeyStr)

  log('Initiator public key found')

  let privateDirectory = `${directoryName}.${initiatorPublicKeyStr}`
  if (hashEncode)
    privateDirectory = await encodeDirectory(privateDirectory)

  if (privateDirectory == null) {
    log("Invalid private directory")
    return
  }

  log("Sending public key")
  let msg = User.nacl.to_hex(user.publicKey())
  await rpc.directoryAdd(privateDirectory, msg, 'default')

  let nextDirectory = await encodeDirectory(user.boxShared(initiatorPublicKey))
  if (nextDirectory == null) {
    log('Invalid next directory (start)')
    return
  }

  let payload = ''

  log('Starting exchange loop...')
  let counter = 1

  while (numIter > 0) {
    // Query
    payload = await rpc.waitAndRemove(nextDirectory, 10)
    if (payload == null) {
      log('Invalid payload (query)')
      return
    }
    nextDirectory = await encodeDirectory(payload)
    if (nextDirectory == null) {
      log("Invalid next_directory (query)", next_directory)
      return
    }

    const message = user.boxDecrypt(payload, initiatorPublicKey)
    if (message == null) {
      log('Invalid query')
      return
    }

    log(`Received: ${message}`)

    // Response
    log('Replying: Pong')
    let msg = `Pong ${counter} ${Date.now()}`
    payload = user.boxEncrypt(msg, initiatorPublicKey)
    if (payload == null) {
      log('Invalid payload (reply)')
      return
    }
    await rpc.directoryAdd(nextDirectory, payload, 'short')

    nextDirectory = await encodeDirectory(payload)
    if (nextDirectory == null) {
      log('Invalid next directory (response)')
      return
    }

    counter += 1
    numIter -= 1
  }
}
