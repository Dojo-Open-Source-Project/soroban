 

class RPCTimeoutError extends Error {
  constructor(message) {
    super(message)
    this.name = "RPCTimeoutError"
  }
}

class RPCCallError extends Error {
  constructor(message) {
    super(message)
    this.name = "RPCCallError"
  }
}

async function encodeDirectory(name) {
  if (name == undefined || name == null || name == '')
    throw Error('encodeDirectory Invalid name')

  const encoder = new TextEncoder('utf-8')
  const utf8Name = encoder.encode(name)
  // Hash the name (sha256)
  const hashBuffer = await window.crypto.subtle.digest('SHA-256', utf8Name)
  // Convert the buffer to a byte array
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  // Convert the bytes to a hex string
  const hashHex = hashArray
    .map((b) => b.toString(16).padStart(2, "0"))
    .join(""); 
  return hashHex;
}

function delay(milliseconds){
  return new Promise(resolve => {
    setTimeout(resolve, milliseconds);
  })
}


class RPC {

    /**
     * Constructor
     * @constructor
     * @param {string | null} url
     * @param {int | null | undefined} timeout
     */
    constructor(url, timeout=15000) {
        this.url = url
        this.timeout = timeout
    }

    /**
     * Send a POST request to the RPC API of a Soroban node 
     * @param {string} method 
     * @param {*} args 
     * @returns 
     */
    async _call(method, args) {
      try {
        const payload = {
          'method': method,
          'params': [args],
          'jsonrpc': '2.0',
          'id': 1
        }

        const parameters = {
          'method': 'POST',
          'mode': 'cors',
          'cache': 'no-cache',
          'credentials': 'omit',
          'timeout': this.timeout,
          'headers': {
            'content-type': 'application/json',
            'User-Agent': 'HotJava/1.1.2 FCS'
          },
          'body': JSON.stringify(payload)
        }

        const response = await fetch(this.url, parameters)

        if (!response.ok) 
          throw new Error(`Received invalid status from RPC API: ${response.status}`)

        const responseObj = await response.json()

        return Object.hasOwn(responseObj, 'result') ? responseObj.result : null

      } catch (e) {
        throw new RPCCallError(`RPC error: ${method}: ${e.cause}`)
      }
    }

    /**
     * Retrieve list of entries stored in Directory for a given key
     * @param {string} name 
     */
    async directoryList(name) {
      const resp = await this._call('directory.List', {'Name': name, 'Entries': []})
      return (resp != null && Object.hasOwn(resp, 'Entries')) ? resp.Entries : []
    }

    /**
     * Store a new entry associated to a given key in the directory 
     * @param {*} name 
     * @param {*} entry 
     * @param {*} mode 
     */
    async directoryAdd(name, entry, mode='default') {
      const resp = await this._call('directory.Add', {'Name': name, 'Entry': entry, 'Mode': mode})
      return !(resp == null || !Object.hasOwn(resp, 'Status') || resp.Status != 'success')
    }

    /**
     * Remove an entry from the Directory
     * @param {*} name 
     * @param {*} entry 
     */
    async directoryRemove(name, entry) {
      const resp = await this._call('directory.Remove', {'Name': name, 'Entry': entry})
      return !(resp == null || !Object.hasOwn(resp, 'Status') || resp.Status != 'success')
    }

    /**
     * Wait for an entry associated to a given key
     * @param {*} directory 
     * @param {*} count 
     */
    async waitAndRemove(directory, count=25) {
      let values = [],
          total = count
      
      count = 0

      while (count < total) {
        values = await this.directoryList(directory)
        count += 1
        if (values.length > 0 || count >= total) {
          break
        }
        // Wait for next list
        await delay(200)
      }

      if (count >= total) {
        throw new RPCTimeoutError(`Wait on ${directory}`)
      }

      // Consider last entry
      const value = values[values.length - 1]
      await this.directoryRemove(directory, value)
      return value
    }

}
