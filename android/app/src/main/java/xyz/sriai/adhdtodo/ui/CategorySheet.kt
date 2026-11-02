package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Category
import xyz.sriai.adhdtodo.data.CategoryInput
import xyz.sriai.adhdtodo.data.IconLibrary
import xyz.sriai.adhdtodo.data.LibraryIcon
import xyz.sriai.adhdtodo.ui.theme.Dims
import xyz.sriai.adhdtodo.ui.theme.Glyphs
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelGrid
import xyz.sriai.adhdtodo.ui.theme.PixelIcon

/** What the Categories page and the category sheet ask the core to do; the defaults do nothing. */
data class CategoryActions(
    val create: suspend (CategoryInput) -> Category = { Category() },
    val update: suspend (id: Long, input: CategoryInput) -> Category = { _, _ -> Category() },
    /** Returns the snapshot [restore] takes. */
    val delete: suspend (id: Long) -> String = { "" },
    val restore: suspend (snapshot: String) -> Unit = {},
    val move: suspend (id: Long, index: Int) -> Unit = { _, _ -> },
    val icons: suspend (query: String) -> IconLibrary = { IconLibrary() },
)

/** internal/domain.CategoryColors: the eight colors in the project folders' order. */
val categoryColorKeys = listOf("indigo", "yellow", "green", "red", "blue", "pink", "teal", "orange")

private val colorNames = listOf(
    R.string.category_color_indigo, R.string.category_color_yellow, R.string.category_color_green,
    R.string.category_color_red, R.string.category_color_blue, R.string.category_color_pink,
    R.string.category_color_teal, R.string.category_color_orange,
)

private val groupNames = mapOf(
    "Work" to R.string.category_group_work, "Study" to R.string.category_group_study,
    "Health" to R.string.category_group_health, "Money" to R.string.category_group_money,
    "Home" to R.string.category_group_home, "Hobby" to R.string.category_group_hobby,
    "Travel" to R.string.category_group_travel, "People" to R.string.category_group_people,
)

/** The 12 x 12 swatch circle of a category's color, drawn at 18 dp like the project folders. */
@Composable
fun PixelSwatch(color: Color, size: Dp = Dims.icon) {
    PixelGrid(size = size) { cell ->
        for (y in Glyphs.disc.indices) {
            for (x in Glyphs.disc[y].indices) {
                if (Glyphs.disc[y][x] == '#') drawRect(color, Offset(x * cell, y * cell), Size(cell, cell))
            }
        }
    }
}

/** A library icon's 12 x 12 picture: `#` pixels in [ink], palette letters in their fixed colors. */
@Composable
fun LibraryPicture(icon: List<List<Int>>, ink: Color, size: Dp = Dims.icon) {
    val palette = LocalTokens.current.categoryPalette
    PixelGrid(size = size) { cell ->
        for (y in icon.indices) {
            for (x in icon[y].indices) {
                val pixel = icon[y][x]
                if (pixel == 0) continue
                val color = if (pixel in 97..108) palette[pixel - 97] else ink
                drawRect(color, Offset(x * cell, y * cell), Size(cell, cell))
            }
        }
    }
}

/**
 * The category sheet: the name (Work and Life keep theirs), the preview at 36, 18 and 12 dp, the
 * color strip, and the icon library with its search. Delete, in blocked red, closes the sheet and
 * hands the snapshot to [onDeleted] for the Undo.
 */
@Composable
fun CategorySheet(
    category: Category?,
    actions: CategoryActions,
    onDismiss: () -> Unit,
    onSaved: (Category) -> Unit = {},
    onDeleted: (Category, snapshot: String) -> Unit = { _, _ -> },
    initialLibrary: IconLibrary? = null,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val scope = rememberCoroutineScope()
    val builtin = category?.builtin?.isNotEmpty() == true
    val original = category?.name.orEmpty()
    var name by remember { mutableStateOf(TextFieldValue(original, TextRange(original.length))) }
    var colorKey by remember { mutableStateOf<String?>(category?.color?.takeIf { it.isNotEmpty() }) }
    var picked by remember { mutableStateOf<LibraryIcon?>(null) }
    var query by remember { mutableStateOf(TextFieldValue("")) }
    var library by remember { mutableStateOf(initialLibrary ?: IconLibrary()) }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var asking by remember { mutableStateOf(false) }

    LaunchedEffect(query.text) {
        if (initialLibrary == null) library = runCatching { actions.icons(query.text.trim()) }.getOrDefault(IconLibrary())
    }

    val shownName = when (category?.builtin) {
        "work" -> stringResource(R.string.category_work)
        "life" -> stringResource(R.string.category_life)
        else -> name.text
    }
    // A new category's color follows its name until a swatch is tapped.
    val effectiveColor = colorKey ?: categoryColorKeys[hashIndex(name.text.trim())]
    val firstIcon = library.groups.firstOrNull()?.icons?.firstOrNull()
    val previewIcon = picked?.icon ?: category?.icon?.takeIf { it.isNotEmpty() } ?: firstIcon?.icon.orEmpty()
    val ink = tokens.categoryColor(effectiveColor)
    val ready = !busy && (builtin || name.text.isNotBlank())

    fun save() {
        if (!ready) return
        busy = true
        scope.launch {
            try {
                val saved = if (category == null) {
                    actions.create(CategoryInput(name = name.text.trim(), color = colorKey, icon = picked?.name))
                } else {
                    actions.update(
                        category.id,
                        CategoryInput(
                            name = if (builtin) null else name.text.trim().takeIf { it != original },
                            color = colorKey?.takeIf { it != category.color },
                            icon = picked?.name,
                        ),
                    )
                }
                onSaved(saved)
                onDismiss()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                error = e.message
                busy = false
            }
        }
    }

    SheetFrame(onDismiss) {
        SheetTitle(stringResource(if (category == null) R.string.category_sheet_new else R.string.category_sheet_edit))
        SectionLabel(stringResource(R.string.category_field_name))
        if (builtin) {
            Text(
                shownName,
                style = type.input,
                color = tokens.textSecondary,
                modifier = Modifier.padding(horizontal = 20.dp, vertical = 8.dp),
            )
            SheetNote(stringResource(R.string.category_builtin_keeps_name))
        } else {
            SheetField(name, { name = it; error = null }, stringResource(R.string.category_field_name), onDone = ::save, focus = category == null)
        }
        error?.let { SheetNote(it) }

        SectionLabel(stringResource(R.string.category_field_icon))
        Row(
            Modifier
                .padding(horizontal = 20.dp)
                .fillMaxWidth()
                .background(tokens.bgSubtle, RoundedCornerShape(12.dp))
                .padding(14.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            for (size in listOf(36.dp, 18.dp, 12.dp)) LibraryPicture(previewIcon, ink, size)
        }

        SectionLabel(stringResource(R.string.category_field_color))
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 14.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            categoryColorKeys.forEachIndexed { index, key ->
                val label = stringResource(R.string.a11y_category_color, stringResource(colorNames[index]))
                Box(
                    Modifier
                        .size(36.dp)
                        .then(
                            if (key == effectiveColor) {
                                Modifier.border(2.dp, tokens.text, RoundedCornerShape(18.dp))
                            } else {
                                Modifier
                            },
                        )
                        .clickable { colorKey = key }
                        .semantics { contentDescription = label },
                    contentAlignment = Alignment.Center,
                ) { PixelSwatch(tokens.categoryColor(key)) }
            }
        }

        SheetField(query, { query = it }, stringResource(R.string.category_search_icons), onDone = {}, focus = false, modifier = Modifier.padding(top = 8.dp))
        IconGrid(library, picked?.name, ink, onPick = { picked = it })

        PrimaryButton(
            label = stringResource(if (category == null) R.string.manage_create else R.string.manage_save),
            enabled = ready,
            onClick = ::save,
            modifier = Modifier.fillMaxWidth().padding(start = 16.dp, end = 16.dp, top = 14.dp),
        )
        if (category != null && !builtin) {
            SheetRow(
                label = stringResource(R.string.category_delete),
                icon = "trash",
                danger = true,
                onClick = { asking = true },
            )
        }
    }
    if (asking && category != null) {
        DeleteDialog(
            title = stringResource(R.string.category_delete_ask_title, category.name),
            body = stringResource(R.string.category_delete_ask_body),
            onConfirm = {
                asking = false
                scope.launch {
                    try {
                        onDeleted(category.copy(), actions.delete(category.id))
                        onDismiss()
                    } catch (e: CancellationException) {
                        throw e
                    } catch (e: Exception) {
                        error = e.message
                    }
                }
            },
            onDismiss = { asking = false },
        )
    }
}

@Composable
private fun SectionLabel(text: String) {
    Text(
        text,
        style = LocalAppType.current.section,
        color = LocalTokens.current.textSecondary,
        modifier = Modifier.padding(start = 20.dp, end = 20.dp, top = 14.dp, bottom = 6.dp),
    )
}

/** The library: curated groups with their headings, or the matches as one flat grid; 240 dp tall. */
@Composable
private fun IconGrid(library: IconLibrary, chosen: String?, ink: Color, onPick: (LibraryIcon) -> Unit) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val empty = library.groups.all { it.icons.isEmpty() }
    Box(
        Modifier
            .padding(horizontal = 16.dp, vertical = 4.dp)
            .fillMaxWidth()
            .height(240.dp)
            .border(Dims.hairline, tokens.border, RoundedCornerShape(12.dp)),
    ) {
        if (empty) {
            Text(
                stringResource(R.string.category_no_icon_match),
                style = type.body,
                color = tokens.textTertiary,
                modifier = Modifier.padding(16.dp),
            )
            return@Box
        }
        LazyVerticalGrid(
            columns = GridCells.Fixed(9),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(6.dp),
            modifier = Modifier.fillMaxWidth(),
        ) {
            for (group in library.groups) {
                if (group.name.isNotEmpty()) {
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        Text(
                            groupNames[group.name]?.let { stringResource(it) } ?: group.name,
                            style = type.meta,
                            color = tokens.textTertiary,
                            modifier = Modifier.padding(start = 6.dp, top = 8.dp, bottom = 4.dp),
                        )
                    }
                }
                items(group.icons, key = { group.name + it.name }) { icon ->
                    val on = icon.name == chosen
                    Box(
                        Modifier
                            .height(38.dp)
                            .then(
                                if (on) {
                                    Modifier
                                        .background(tokens.accent.copy(alpha = 0.10f), RoundedCornerShape(8.dp))
                                        .border(2.dp, tokens.accent, RoundedCornerShape(8.dp))
                                } else {
                                    Modifier
                                },
                            )
                            .clickable { onPick(icon) }
                            .semantics { contentDescription = icon.name },
                        contentAlignment = Alignment.Center,
                    ) { LibraryPicture(icon.icon, if (on) ink else tokens.text) }
                }
            }
        }
    }
}

/** domain.DefaultCategoryColor's index: FNV-1a (32 bit) of the name's UTF-8 bytes, modulo eight. */
internal fun hashIndex(name: String): Int {
    var hash = -2128831035
    for (byte in name.toByteArray(Charsets.UTF_8)) {
        hash = (hash xor (byte.toInt() and 0xff)) * 16777619
    }
    return ((hash.toLong() and 0xffffffffL) % categoryColorKeys.size).toInt()
}

/** The row `New category…` that ends the task menu's Category page and the Categories list. */
@Composable
fun NewCategoryRow(onClick: () -> Unit) {
    val tokens = LocalTokens.current
    SheetRow(
        label = stringResource(R.string.category_new),
        leading = { PixelIcon("plus", size = 20.dp, tint = tokens.accentInk) },
        onClick = onClick,
    )
}
