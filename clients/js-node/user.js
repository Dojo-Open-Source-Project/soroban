import nacl_factory from 'js-nacl'


export class User {

    /**
     * NACL instance
     */
    static nacl = null

    /**
     * Constructor
     */
    constructor() {
      this.keyPair = null
    }

    /**
     * Instantiate NACL
     */
    static async instantiateNacl() {
      if (User.nacl == null)
        User.nacl = await nacl_factory.instantiate(nacl => {})
    }

    /**
     * Generate a new User object
     * @returns {User}
     */
    static async createNewUser() {
      let user = new User()
      await User.instantiateNacl()
      user.keyPair = User.nacl.crypto_box_keypair() 
      return user
    }

    /**
     * Return User's private key 
     * @returns {UInt8Array}
     */
    privateKey() {
      return this.keyPair.boxSk
    }

    /**
     * Return User's public key
     * @returns {UInt8Array}
     */
    publicKey() {
      return this.keyPair.boxPk
    }

    /**
     * Encrypt a message with a shared secret 
     * generated for a given public key
     * @param {string} msg 
     * @param {UInt8Array} pubKeyB  
     * @returns {String}
     */
    boxEncrypt(msg, pubKeyB) {
      if (this.keyPair == null)
        throw new Error('boxEncrypt Box not initialized')
      if (msg == undefined || msg == null)
        throw new Error('boxEncrypt Invalid message')

      const encodedMsg = User.nacl.encode_utf8(msg)
      const nonce = User.nacl.crypto_box_random_nonce()

      const encrypted = User.nacl.crypto_box(
        encodedMsg, 
        nonce, 
        pubKeyB, 
        this.privateKey()
      )

      // Concatenate the nonce and the encrypted message
      const mergedArray = new Uint8Array(nonce.length + encrypted.length);
      mergedArray.set(nonce);
      mergedArray.set(encrypted, nonce.length);

      return User.nacl.to_hex(mergedArray)
    }

    /**
     * Decrypt a message encrypted with a shared secret 
     * generated for a given public key
     * @param {string} data
     * @param {UInt8Array} pubkeyA 
     * @returns {String}
     */
    boxDecrypt(data, pubkeyA) {
      if (this.keyPair == null)
        throw new Error('boxEncrypt Box not initialized')
      if (data == undefined || data == null)
        throw new Error('boxDecrypt Invalid data packet')

      data = User.nacl.from_hex(data)
      const nonce = data.slice(0, 24)
      const msg = data.slice(24, data.length)
      
      const encodedMsg = User.nacl.crypto_box_open(
        msg, 
        nonce, 
        pubkeyA, 
        this.privateKey()
      )
      
      return User.nacl.decode_utf8(encodedMsg)
    }

    /**
     * Generate a secret shared with a given public key
     * @param {UInt8Array} pubKeyB 
     * @returns {String}
     */
    boxShared(pubKeyB) {
      if (this.keyPair == null)
        throw new Error('boxEncrypt Box not initialized')

      const secret = User.nacl.crypto_box_precompute(
        pubKeyB, 
        this.privateKey()
      )

      return User.nacl.to_hex(secret.boxK)
    }
}
