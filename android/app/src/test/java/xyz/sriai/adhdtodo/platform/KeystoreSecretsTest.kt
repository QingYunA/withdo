package xyz.sriai.adhdtodo.platform

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Exercises KeystoreSecrets' storage logic with a fake cipher, so the JVM test needs neither
 * AndroidKeyStore nor SharedPreferences.
 */
class KeystoreSecretsTest {

    private class MemoryStorage : SecretStorage {
        val values = mutableMapOf<String, String>()

        override fun get(key: String): String? = values[key]

        override fun put(key: String, value: String) {
            values[key] = value
        }

        override fun remove(key: String): Boolean = values.remove(key) != null
    }

    private class FakeCipher : SecretCipher {
        var encryptions = 0

        override fun encrypt(plaintext: String): String {
            encryptions++
            return "$encryptions:$plaintext"
        }

        override fun decrypt(payload: String): String = payload.substringAfter(':')
    }

    @Test
    fun setGetOverwriteAndDelete() {
        val storage = MemoryStorage()
        val cipher = FakeCipher()
        val secrets = KeystoreSecrets(storage, cipher)

        secrets.set("sync/refresh-token", "first")
        assertEquals("first", secrets.get("sync/refresh-token"))
        assertEquals(1, cipher.encryptions)

        secrets.set("sync/refresh-token", "second")
        assertEquals("second", secrets.get("sync/refresh-token"))
        assertEquals(2, cipher.encryptions)

        secrets.delete("sync/refresh-token")
        assertFalse("sync/refresh-token" in storage.values)
        assertEquals(2, cipher.encryptions)
    }

    @Test
    fun storedValuesAreCiphertext() {
        val storage = MemoryStorage()
        val secrets = KeystoreSecrets(storage, FakeCipher())

        secrets.set("sync/refresh-token", "a secret")

        val payload = storage.values.getValue("sync/refresh-token")
        assertNotEquals("a secret", payload)
        assertTrue(payload.startsWith("1:"))
    }

    @Test
    fun missingKeySaysNotFound() {
        val secrets = KeystoreSecrets(MemoryStorage(), FakeCipher())

        val missing = assertThrows(NoSuchElementException::class.java) {
            secrets.get("sync/refresh-token")
        }
        assertEquals("not found", missing.message)

        val deleted = assertThrows(NoSuchElementException::class.java) {
            secrets.delete("sync/refresh-token")
        }
        assertEquals("not found", deleted.message)
    }
}
