package xyz.sriai.adhdtodo.ui

import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.VectorConverter
import androidx.compose.animation.core.animateValue
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.pullToRefresh
import androidx.compose.material3.pulltorefresh.rememberPullToRefreshState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.SyncStatus
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon

/** How far a list is pulled before it syncs, and the gap it holds open while syncing (v43 P2). */
val PullGap: Dp = 64.dp

/** A pull syncs only on a phone that is signed in and linked; otherwise the list just springs back. */
fun pullSyncs(status: SyncStatus): Boolean = status.signedIn && status.linked

/**
 * A list that syncs when pulled down past [PullGap]: the list follows the finger, and the pixel
 * cloud with `Syncing…` sits in the gap above it until [syncing] ends. With [enabled] false the
 * pull is the system's overscroll. [preview] holds the gap open, for the design screenshots.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PullToSync(
    enabled: Boolean,
    syncing: Boolean,
    onSync: () -> Unit,
    modifier: Modifier = Modifier,
    preview: Boolean = false,
    content: @Composable () -> Unit,
) {
    val state = rememberPullToRefreshState()
    val open = syncing || preview
    Box(
        modifier
            .clipToBounds()
            .pullToRefresh(isRefreshing = syncing, state = state, enabled = enabled, threshold = PullGap, onRefresh = onSync),
    ) {
        val fraction = if (preview) 1f else state.distanceFraction.coerceIn(0f, 1.5f)
        if (fraction > 0f) PullIndicator(fraction, open, Modifier.height(PullGap * fraction))
        Box(Modifier.graphicsLayer { translationY = PullGap.toPx() * fraction }) { content() }
    }
}

/** The gap's content: the 24 dp cloud, faded in with the pull, and `Syncing…` under it once it runs. */
@Composable
private fun PullIndicator(fraction: Float, syncing: Boolean, modifier: Modifier) {
    val tokens = LocalTokens.current
    Column(
        modifier.fillMaxWidth(),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(4.dp, Alignment.CenterVertically),
    ) {
        val lift = if (syncing) cloudBob() else 0
        PixelIcon(
            "cloud",
            size = 24.dp,
            tint = tokens.textTertiary,
            modifier = Modifier
                .alpha(fraction.coerceAtMost(1f))
                .graphicsLayer { translationY = -lift * 2.dp.toPx() },
        )
        if (syncing) {
            Text(stringResource(R.string.pull_syncing), style = LocalAppType.current.meta, color = tokens.textSecondary, maxLines = 1)
        }
    }
}

/** The cloud bobs one pixel (2 dp) up and down in 300 ms steps while the sync runs: motion in pixels. */
@Composable
private fun cloudBob(): Int {
    val transition = rememberInfiniteTransition(label = "pull-cloud")
    val step by transition.animateValue(
        initialValue = 0,
        targetValue = 2,
        typeConverter = Int.VectorConverter,
        animationSpec = infiniteRepeatable(tween(600, easing = LinearEasing), RepeatMode.Restart),
        label = "pull-cloud-step",
    )
    return step % 2
}
