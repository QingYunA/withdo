package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon

enum class Tab(val label: Int, val icon: String) {
    MINE(R.string.nav_mine, ""),
    AGENTS(R.string.nav_agents, "robot"),
    CALENDAR(R.string.nav_calendar, "calendar"),
    PROJECTS(R.string.nav_projects, "folder"),
}

/**
 * The sidebar becomes the phone's bottom bar: 18 dp icon over a 12 sp label, 64 dp tall, on
 * bg.sidebar with a 0.5 dp line above it. The selected tab sits on the neutral bg.selected pill.
 */
@Composable
fun BottomNav(tab: Tab, count: Int, onSelect: (Tab) -> Unit) {
    val tokens = LocalTokens.current
    Column(Modifier.fillMaxWidth().background(tokens.bgSidebar).navigationBarsPadding()) {
        Box(Modifier.fillMaxWidth().height(Dims.hairline).background(tokens.border))
        Row(Modifier.fillMaxWidth().height(Dims.navHeight)) {
            for (entry in Tab.entries) {
                NavItem(
                    tab = entry,
                    selected = entry == tab,
                    count = if (entry == Tab.MINE) count else 0,
                    modifier = Modifier.weight(1f),
                    onSelect = { onSelect(entry) },
                )
            }
        }
    }
}

@Composable
private fun NavItem(tab: Tab, selected: Boolean, count: Int, modifier: Modifier, onSelect: () -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val ink = if (selected) tokens.text else tokens.textSecondary
    val iconInk = if (selected) tokens.text else tokens.textSecondary
    Column(
        modifier = modifier.fillMaxHeight().clickable { onSelect() },
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Box(
            modifier = Modifier
                .width(56.dp)
                .height(28.dp)
                .background(if (selected) tokens.bgSelected else Color.Transparent, RoundedCornerShape(14.dp)),
            contentAlignment = Alignment.Center,
        ) {
            if (tab == Tab.MINE) {
                Avatar("you", size = Dims.avatar)
            } else {
                PixelIcon(tab.icon, size = Dims.icon, tint = iconInk)
            }
            if (count > 0) {
                Text(
                    if (count > 99) "99+" else count.toString(),
                    style = type.badge,
                    color = Color.White,
                    textAlign = TextAlign.Center,
                    modifier = Modifier
                        .align(Alignment.TopEnd)
                        .offset(x = (-4).dp, y = (-2).dp)
                        .height(18.dp)
                        .background(tokens.statusBlocked, CircleShape)
                        .padding(horizontal = 5.dp),
                )
            }
        }
        Text(
            stringResource(tab.label),
            style = if (selected) type.navSelected else type.nav,
            color = ink,
            modifier = Modifier.padding(top = 4.dp),
        )
    }
}
