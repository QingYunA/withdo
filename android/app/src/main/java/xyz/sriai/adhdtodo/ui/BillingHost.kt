package xyz.sriai.adhdtodo.ui

import android.content.Context
import android.content.ContextWrapper
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.LifecycleOwner
import kotlinx.coroutines.launch
import xyz.sriai.adhdtodo.data.BillingBackend

/**
 * Keeps [holder] up to date and draws its sheet. The plan is read again whenever [refreshKey]
 * changes (the sync status carries a counter the core raises when the plan changed, so the
 * `billing` event reaches here), and again from the server when the app comes back to the screen
 * while the sheet waits for a payment or Settings is open. [openCheckout] opens a checkout URL in
 * the browser (a Custom Tab).
 */
@Composable
fun BillingHost(
    holder: BillingHolder,
    backend: BillingBackend,
    refreshKey: Any?,
    onSettings: Boolean,
    openCheckout: (String) -> Unit,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val lang = checkoutLanguage(context.resources)
    LaunchedEffect(refreshKey, onSettings) { holder.read(backend) }

    val refreshOnResume = rememberUpdatedState(holder.stage == SheetStage.WAITING || onSettings)
    val owner = remember(context) { context.lifecycleOwner() }
    DisposableEffect(owner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME && refreshOnResume.value) scope.launch { holder.refresh(backend) }
        }
        owner?.lifecycle?.addObserver(observer)
        onDispose { owner?.lifecycle?.removeObserver(observer) }
    }

    if (holder.stage != SheetStage.CLOSED) {
        UpgradeSheet(
            holder = holder,
            onStart = { provider -> scope.launch { holder.start(backend, lang, provider, openCheckout) } },
            onReopen = { openCheckout(holder.checkoutUrl) },
            onDismiss = holder::close,
        )
    }
}

private tailrec fun Context.lifecycleOwner(): LifecycleOwner? = when (this) {
    is LifecycleOwner -> this
    is ContextWrapper -> baseContext.lifecycleOwner()
    else -> null
}
