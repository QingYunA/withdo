package xyz.sriai.adhdtodo.ui

import android.util.Log
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import xyz.sriai.adhdtodo.data.CoreApi
import xyz.sriai.adhdtodo.data.Task

/**
 * The Search route: it owns the text, asks the core a moment after typing stops, and asks again
 * when the data changes ([revision], [tick]) so a reopened task leaves or joins the list at once.
 * A failed read keeps the last answer on screen.
 */
@Composable
fun SearchRoute(
    api: CoreApi,
    words: Words,
    revision: Int,
    tick: Int,
    query: String,
    onQueryChange: (String) -> Unit,
    onCancel: () -> Unit,
    onOpenTask: (Long) -> Unit,
    onLongPress: (Long) -> Unit,
    onToggle: (Task) -> Unit,
) {
    var results by remember { mutableStateOf<SearchResults?>(null) }
    LaunchedEffect(query, revision, tick) {
        val text = query.trim()
        if (text.isEmpty()) {
            results = null
            return@LaunchedEffect
        }
        delay(120)
        try {
            results = withContext(Dispatchers.IO) {
                val tasks = api.search(text)
                val parents = parentIds(tasks).mapNotNull { id ->
                    runCatching { api.task(id).task.title }.getOrNull()?.let { id to it }
                }.toMap()
                groupSearch(text, tasks, api.projects(), parents)
            }
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            Log.w("adhd-todo", "search failed", e)
        }
    }
    SearchScreen(
        query = query,
        onQueryChange = onQueryChange,
        results = results,
        words = words,
        onCancel = onCancel,
        onOpenTask = onOpenTask,
        onLongPress = onLongPress,
        onToggle = onToggle,
    )
}
