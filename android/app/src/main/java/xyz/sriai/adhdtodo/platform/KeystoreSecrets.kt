package xyz.sriai.adhdtodo.platform

import android.content.Context
import android.content.SharedPreferences
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import xyz.sriai.adhdtodo.core.Secrets
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

/** The private key/value storage behind [KeystoreSecrets], small enough to fake in a JVM test. */
interface SecretStorage {
    fun get(key: String): String?
    fun put(key: String, value: String)
    fun remove(key: String): Boolean
}

/** The encryption behind [KeystoreSecrets], small enough to fake in a JVM test. */
interface SecretCipher {
    fun encrypt(plaintext: String): String
    fun decrypt(payload: String): String
}

/**
 * The phone's [Secrets]: each value is encrypted with a fresh AES-GCM IV and kept as base64 in a
 * private SharedPreferences file. The AES-256 key lives in AndroidKeyStore under [KEY_ALIAS].
 *
 * The storage and cipher are constructor parameters so the JVM test can exercise set, get,
 * overwrite, delete and the "not found" message without an Android Keystore.
 */
class KeystoreSecrets(
    private val storage: SecretStorage,
    private val cipher: SecretCipher,
) : Secrets {

    constructor(context: Context) : this(
        SharedPreferencesStorage(context.getSharedPreferences(FILE_NAME, Context.MODE_PRIVATE)),
        KeystoreCipher(),
    )

    override fun get(key: String): String {
        val payload = storage.get(key) ?: throw NoSuchElementException(NOT_FOUND)
        return cipher.decrypt(payload)
    }

    override fun set(key: String, value: String) {
        storage.put(key, cipher.encrypt(value))
    }

    override fun delete(key: String) {
        if (!storage.remove(key)) {
            throw NoSuchElementException(NOT_FOUND)
        }
    }

    private companion object {
        const val FILE_NAME = "adhd-todo-secrets"
        const val NOT_FOUND = "not found"
    }
}

/** SharedPreferences with synchronous writes, so the Go side sees a value as soon as set returns. */
private class SharedPreferencesStorage(private val preferences: SharedPreferences) : SecretStorage {
    override fun get(key: String): String? = preferences.getString(key, null)

    override fun put(key: String, value: String) {
        if (!preferences.edit().putString(key, value).commit()) {
            throw IllegalStateException("could not store the value")
        }
    }

    override fun remove(key: String): Boolean {
        if (!preferences.contains(key)) {
            return false
        }
        return preferences.edit().remove(key).commit()
    }
}

/** AES-256-GCM with a key in AndroidKeyStore; every encrypt uses a fresh IV from the cipher. */
private class KeystoreCipher : SecretCipher {
    override fun encrypt(plaintext: String): String {
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.ENCRYPT_MODE, key())
        val encrypted = cipher.doFinal(plaintext.toByteArray(Charsets.UTF_8))
        return Base64.encodeToString(cipher.iv + encrypted, Base64.NO_WRAP)
    }

    override fun decrypt(payload: String): String {
        val raw = Base64.decode(payload, Base64.NO_WRAP)
        if (raw.size <= IV_SIZE) {
            throw IllegalStateException("the stored value is damaged")
        }
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.DECRYPT_MODE, key(), GCMParameterSpec(TAG_BITS, raw, 0, IV_SIZE))
        return String(cipher.doFinal(raw, IV_SIZE, raw.size - IV_SIZE), Charsets.UTF_8)
    }

    @Synchronized
    private fun key(): SecretKey {
        val store = KeyStore.getInstance(KEYSTORE).apply { load(null) }
        val existing = store.getEntry(KEY_ALIAS, null) as? KeyStore.SecretKeyEntry
        if (existing != null) {
            return existing.secretKey
        }
        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, KEYSTORE)
        generator.init(
            KeyGenParameterSpec.Builder(KEY_ALIAS, KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT)
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setKeySize(256)
                .build(),
        )
        return generator.generateKey()
    }

    private companion object {
        const val KEYSTORE = "AndroidKeyStore"
        const val KEY_ALIAS = "adhd-todo-secrets"
        const val TRANSFORMATION = "AES/GCM/NoPadding"
        const val TAG_BITS = 128
        const val IV_SIZE = 12
    }
}
