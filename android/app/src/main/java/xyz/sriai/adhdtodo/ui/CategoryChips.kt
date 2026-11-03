package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.ui.theme.CategoryIcon
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens

/**
 * Today's category chips (v43), under the date: `All`, then each category with its 12 dp icon and
 * name, 32 dp tall, scrolling sideways when they overflow. The chosen chip is on `bg.selected`; the
 * others have a 0.5 dp edge. [selected] 0 is All.
 */
@Composable
fun CategoryChips(
    chips: List<Category>,
    selected: Long,
    words: Words,
    onSelect: (Long) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState())
            .padding(start = 16.dp, end = 16.dp, top = 4.dp, bottom = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Chip(selected = selected == 0L, label = stringResource(R.string.chips_all), description = null) { onSelect(0L) }
        for (category in chips) {
            val name = words.categoryName(category)
            Chip(
                selected = selected == category.id,
                label = name,
                description = stringResource(R.string.chips_show_only, name),
                icon = { CategoryIcon(category, size = 12.dp) },
            ) { onSelect(category.id) }
        }
    }
}

@Composable
private fun Chip(
    selected: Boolean,
    label: String,
    description: String?,
    icon: (@Composable () -> Unit)? = null,
    onClick: () -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val shape = RoundedCornerShape(16.dp)
    Row(
        modifier = Modifier
            .height(32.dp)
            .clip(shape)
            .then(
                if (selected) Modifier.background(tokens.bgSelected, shape)
                else Modifier.border(Dims.hairline, tokens.border, shape),
            )
            .clickable(role = Role.Tab, onClick = onClick)
            .semantics {
                this.selected = selected
                if (description != null) contentDescription = description
            }
            .padding(horizontal = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        icon?.invoke()
        Text(
            label,
            style = type.meta,
            color = if (selected) tokens.text else tokens.textSecondary,
            maxLines = 1,
        )
    }
}
