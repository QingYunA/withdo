package xyz.sriai.adhdtodo.platform

import android.content.Context
import java.time.ZonedDateTime
import java.util.TimeZone
import xyz.sriai.adhdtodo.core.Core
import xyz.sriai.adhdtodo.core.Core_

/**
 * Every place that opens the Go core goes through here. Go on Android has no local time zone (its
 * time.Local is UTC), so the phone's zone is handed over first: "today", "tomorrow at 9" and the
 * calendar follow the phone, not UTC.
 */
object CoreOpen {
    fun open(context: Context): Core_ {
        setZone()
        return Core.open(context.filesDir.absolutePath, KeystoreSecrets(context))
    }

    /** Hands the phone's zone to the core again; cheap when nothing changed. */
    fun setZone() {
        val zone = TimeZone.getDefault().id
        val offset = ZonedDateTime.now().offset.totalSeconds.toLong()
        runCatching { Core.setZone(zone, offset) }
    }
}
