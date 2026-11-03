package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectDragGesturesAfterLongPress
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.zIndex
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.CategoryCount
import xyz.sriai.adhdtodo.data.IconLibrary
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon

/** What the category sheet is open on: a new category, or one of the list. */
private sealed interface CategoryTarget {
    data object New : CategoryTarget
    data class Edit(val category: Category) : CategoryTarget
}

/** The row height the drag reads in; a row is a name, a quiet count line and a chevron. */
private val CATEGORY_ROW = 60.dp

/**
 * `Settings > Categories`: the categories in order (swatch, name, `12 tasks`, `›`), a long press
 * drags one to a new place, `New category…` ends the list in accent, and a row opens the category
 * sheet. Deleting offers Undo for six seconds above the bottom bar.
 */
@Composable
fun CategoriesScreen(
    categories: List<CategoryCount>,
    actions: CategoryActions,
    onChanged: () -> Unit,
    libraryForPreview: IconLibrary? = null,
    startWith: Category? = null,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val scope = rememberCoroutineScope()
    val haptic = LocalHapticFeedback.current
    val rowPx = with(LocalDensity.current) { CATEGORY_ROW.toPx() }
    var order by remember(categories) { mutableStateOf(categories) }
    var dragId by remember { mutableStateOf<Long?>(null) }
    var dragY by remember { mutableFloatStateOf(0f) }
    var target by remember { mutableStateOf<CategoryTarget?>(startWith?.let { CategoryTarget.Edit(it) }) }
    var deleted by remember { mutableStateOf<Pair<String, String>?>(null) }

    LaunchedEffect(deleted) {
        if (deleted == null) return@LaunchedEffect
        delay(6000)
        deleted = null
    }

    Box(Modifier.fillMaxSize()) {
        Column(
            Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = Dims.pagePadding)
                .padding(bottom = 80.dp),
        ) {
            PageTitle(stringResource(R.string.settings_categories), stringResource(R.string.categories_line))
            GroupCard {
                for ((index, row) in order.withIndex()) {
                    val id = row.category.id
                    val dragging = dragId == id
                    if (index > 0 && !dragging) RowRule()
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .height(CATEGORY_ROW)
                            .zIndex(if (dragging) 1f else 0f)
                            .graphicsLayer { translationY = if (dragging) dragY else 0f }
                            .then(if (dragging) Modifier.shadow(6.dp).background(tokens.bgSelected) else Modifier)
                            .clickable { target = CategoryTarget.Edit(row.category) }
                            .pointerInput(id) {
                                detectDragGesturesAfterLongPress(
                                    onDragStart = {
                                        dragId = id
                                        dragY = 0f
                                        haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                                    },
                                    onDrag = { change, amount ->
                                        change.consume()
                                        dragY += amount.y
                                        val at = order.indexOfFirst { it.category.id == id }
                                        if (dragY > rowPx / 2 && at < order.lastIndex) {
                                            order = order.swap(at, at + 1)
                                            dragY -= rowPx
                                        } else if (dragY < -rowPx / 2 && at > 0) {
                                            order = order.swap(at, at - 1)
                                            dragY += rowPx
                                        }
                                    },
                                    onDragEnd = {
                                        val to = order.indexOfFirst { it.category.id == id }
                                        val from = categories.indexOfFirst { it.category.id == id }
                                        dragId = null
                                        dragY = 0f
                                        if (to >= 0 && to != from) {
                                            scope.launch {
                                                runCatching { actions.move(id, to) }
                                                onChanged()
                                            }
                                        }
                                    },
                                    onDragCancel = { dragId = null; dragY = 0f },
                                )
                            }
                            .padding(horizontal = 16.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(14.dp),
                    ) {
                        PixelSwatch(tokens.categoryColor(row.category.color), size = 12.dp)
                        Column(Modifier.weight(1f)) {
                            Text(
                                categoryTitle(row.category),
                                style = type.body.copy(fontSize = 16.sp),
                                color = tokens.text,
                                maxLines = 1,
                            )
                            Text(
                                pluralStringResource(R.plurals.settings_category_tasks, row.open, row.open),
                                style = type.meta,
                                color = tokens.textSecondary,
                            )
                        }
                        PixelIcon("chevron-right", size = 12.dp, tint = tokens.textTertiary)
                    }
                }
                if (order.isNotEmpty()) RowRule()
                Row(
                    Modifier
                        .fillMaxWidth()
                        .height(52.dp)
                        .clickable { target = CategoryTarget.New }
                        .padding(horizontal = 16.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(14.dp),
                ) {
                    PixelIcon("plus", size = 14.dp, tint = tokens.accentInk)
                    Text(stringResource(R.string.category_new), style = type.body.copy(fontSize = 16.sp), color = tokens.accentInk)
                }
            }
        }
        deleted?.let { (message, snapshot) ->
            UndoBar(
                message = message,
                onUndo = {
                    deleted = null
                    scope.launch {
                        runCatching { actions.restore(snapshot) }
                        onChanged()
                    }
                },
                modifier = Modifier.align(Alignment.BottomCenter),
            )
        }
    }

    target?.let { current ->
        val editing = (current as? CategoryTarget.Edit)?.category
        val open = categories.firstOrNull { it.category.id == editing?.id }?.open ?: 0
        val unsorted = stringResource(R.string.category_deleted_empty, editing?.name.orEmpty())
        val withCount = pluralStringResource(R.plurals.category_deleted, open, editing?.name.orEmpty(), open)
        CategorySheet(
            category = editing,
            actions = actions,
            initialLibrary = libraryForPreview,
            onDismiss = { target = null },
            onSaved = { onChanged() },
            onDeleted = { _, snapshot ->
                deleted = (if (open > 0) withCount else unsorted) to snapshot
                onChanged()
            },
        )
    }
}

@Composable
private fun categoryTitle(category: Category): String = when (category.builtin) {
    "work" -> stringResource(R.string.category_work)
    "life" -> stringResource(R.string.category_life)
    else -> category.name
}

/** The list with the items at [a] and [b] exchanged. */
internal fun <T> List<T>.swap(a: Int, b: Int): List<T> =
    toMutableList().also { val x = it[a]; it[a] = it[b]; it[b] = x }
