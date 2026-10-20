package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon

/** The page's title and its quiet second line, with the sync cloud at the right. */
@Composable
fun PageHeader(title: String, subtitle: String, syncBlocked: Boolean, onSync: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = Dims.pagePadding, end = Dims.pagePadding, top = 10.dp, bottom = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(title, style = type.title, color = tokens.text)
        Text(
            subtitle,
            style = type.body,
            color = tokens.textTertiary,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.padding(start = 10.dp).weight(1f),
        )
        SyncCloud(blocked = syncBlocked, onSync = onSync)
    }
}

/** The 18 dp pixel cloud that opens Sync; red only for the desktop's two reasons. */
@Composable
fun SyncCloud(blocked: Boolean, onSync: () -> Unit) {
    val tokens = LocalTokens.current
    PixelIcon(
        "cloud",
        size = Dims.icon,
        tint = if (blocked) tokens.statusBlocked else tokens.textTertiary,
        modifier = Modifier.clickable(onClick = onSync).padding(4.dp),
    )
}
