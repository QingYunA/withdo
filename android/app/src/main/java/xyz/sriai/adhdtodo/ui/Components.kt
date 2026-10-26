package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon

/** A section's title in sentence case, with its quiet count. */
@Composable
fun SectionHeader(
    title: String,
    count: Int? = null,
    start: androidx.compose.ui.unit.Dp = 0.dp,
    end: (@Composable () -> Unit)? = null,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = start, top = 22.dp, bottom = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(title, style = type.section, color = tokens.textSecondary)
        if (count != null) Text(count.toString(), style = type.meta, color = tokens.textSecondary)
        if (end != null) {
            Spacer(Modifier.weight(1f))
            end()
        }
    }
}

/** The small outlined P0 or P1 of a row; P2 and P3 are never drawn. */
@Composable
fun PriorityBadge(priority: String) {
    if (priority != "P0" && priority != "P1") return
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val shape = RoundedCornerShape(5.dp)
    Box(
        modifier = Modifier
            .border(Dims.hairline, tokens.priorityRing(priority), shape)
            .padding(horizontal = 5.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(priority, style = type.badge, color = tokens.priorityInk(priority))
    }
}

/** A status pill: 13 sp on a 12 % tint of the status color. */
@Composable
fun StatusPill(status: String, label: String, modifier: Modifier = Modifier) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Box(
        modifier = modifier
            .background(tokens.statusTint(status), RoundedCornerShape(5.dp))
            .padding(horizontal = 6.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(label, style = type.meta, color = tokens.status(status))
    }
}

@Composable
fun PixelRule(modifier: Modifier = Modifier) {
    Box(modifier.fillMaxWidth().background(LocalTokens.current.border))
}

/** The calm picture of an empty screen: you and the assistant, a title and one quiet line. */
@Composable
fun EmptyState(
    title: String,
    body: String,
    modifier: Modifier = Modifier,
    top: Dp = 72.dp,
    link: String? = null,
    onLink: (() -> Unit)? = null,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Column(
        modifier = modifier.fillMaxWidth().padding(top = top),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            xyz.sriai.adhdtodo.ui.theme.Avatar("you", size = 60.dp)
            xyz.sriai.adhdtodo.ui.theme.Avatar("assistant", size = 60.dp)
        }
        Text(
            title,
            style = type.bodyStrong,
            color = tokens.text,
            textAlign = TextAlign.Center,
            modifier = Modifier.padding(top = 14.dp),
        )
        Text(
            body,
            style = type.body,
            color = tokens.textSecondary,
            textAlign = TextAlign.Center,
            modifier = Modifier.widthIn(max = 480.dp),
        )
        if (link != null && onLink != null) {
            Text(
                link,
                style = type.body,
                color = tokens.accentInk,
                modifier = Modifier.padding(top = 6.dp).clickable { onLink() },
            )
        }
    }
}

/** Calendar and Projects are placeholders in A2a; they say what comes and when. */
@Composable
fun PlaceholderScreen(icon: String, title: String, body: String) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    Column(
        modifier = Modifier.fillMaxWidth().padding(top = 72.dp, start = 20.dp, end = 20.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        PixelIcon(icon, size = 36.dp, tint = tokens.textTertiary)
        Text(title, style = type.bodyStrong, color = tokens.text, textAlign = TextAlign.Center)
        Text(
            body,
            style = type.body,
            color = tokens.textSecondary,
            textAlign = TextAlign.Center,
            modifier = Modifier.widthIn(max = 480.dp),
        )
    }
}

@Composable
fun HowToConnectDialog(onDismiss: () -> Unit) {
    val tokens = LocalTokens.current
    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = tokens.bg,
        titleContentColor = tokens.text,
        textContentColor = tokens.textSecondary,
        title = { Text(stringResource(xyz.sriai.adhdtodo.R.string.connect_title)) },
        text = { Text(stringResource(xyz.sriai.adhdtodo.R.string.connect_body)) },
        confirmButton = {
            TextButton(onClick = onDismiss) {
                Text(stringResource(xyz.sriai.adhdtodo.R.string.close), color = tokens.accentInk)
            }
        },
    )
}
