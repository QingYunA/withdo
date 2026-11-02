package xyz.sriai.adhdtodo

import android.content.Intent
import android.os.Bundle
import android.util.Log
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.input.TextFieldValue
import androidx.lifecycle.lifecycleScope
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import xyz.sriai.adhdtodo.core.Core
import xyz.sriai.adhdtodo.core.Core_
import xyz.sriai.adhdtodo.data.CoreApi
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.QuickParse
import xyz.sriai.adhdtodo.platform.AppLanguage
import xyz.sriai.adhdtodo.platform.KeystoreSecrets
import xyz.sriai.adhdtodo.ui.ShareDraft
import xyz.sriai.adhdtodo.ui.ShareForm
import xyz.sriai.adhdtodo.ui.ShareSheet
import xyz.sriai.adhdtodo.ui.ShareWhen
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.ui.createQuickDrafts
import xyz.sriai.adhdtodo.ui.dueStamp
import xyz.sriai.adhdtodo.ui.quickSummary
import xyz.sriai.adhdtodo.ui.readAny
import xyz.sriai.adhdtodo.ui.rememberQuickParse
import xyz.sriai.adhdtodo.ui.shareDraft
import xyz.sriai.adhdtodo.ui.shareWhen
import xyz.sriai.adhdtodo.ui.theme.AdhdTheme
import xyz.sriai.adhdtodo.ui.theme.Appearance
import java.time.LocalDate
import java.time.ZoneId

/**
 * Share into Withdo (v43 P2): the system share sheet's text and links become a task. The Activity
 * is see-through, so its sheet sits over the other app; Add (or a dismiss) finishes it and the
 * other app is back. It opens its own short-lived core, as the reminder receiver does.
 */
class ShareActivity : ComponentActivity() {
    private var core: Core_? = null

    override fun attachBaseContext(newBase: android.content.Context) {
        super.attachBaseContext(AppLanguage.wrap(newBase))
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Appearance.init(this)
        val subject = intent.getStringExtra(Intent.EXTRA_SUBJECT)
        val draft = shareDraft(intent.getStringExtra(Intent.EXTRA_TEXT), subject)
        if (intent.action != Intent.ACTION_SEND || draft == null) {
            finish()
            return
        }
        lifecycleScope.launch {
            val opened = withContext(Dispatchers.IO) {
                runCatching { Core.open(filesDir.absolutePath, KeystoreSecrets(this@ShareActivity)) }.getOrNull()
            }
            if (opened == null) {
                done(R.string.share_failed)
                return@launch
            }
            core = opened
            setContent {
                AdhdTheme {
                    ShareRoute(
                        api = remember { CoreApi(opened) },
                        draft = draft,
                        pageTitle = subject?.trim().orEmpty(),
                        onDone = { message -> done(message) },
                        onDismiss = { finish() },
                    )
                }
            }
        }
    }

    /** Says [message] in a toast over the other app and goes back to it. */
    private fun done(message: Int) {
        Toast.makeText(applicationContext, message, Toast.LENGTH_SHORT).show()
        finish()
    }

    override fun onDestroy() {
        core?.close()
        core = null
        super.onDestroy()
    }
}

/**
 * The sheet with its state: the projects, the parser's reading of the title, and Add, which makes
 * the task (with what the parser read, or the picked day) and keeps the link on it.
 */
@Composable
private fun ShareRoute(
    api: CoreApi,
    draft: ShareDraft,
    pageTitle: String,
    onDone: (Int) -> Unit,
    onDismiss: () -> Unit,
) {
    val resources = LocalContext.current.resources
    val words = remember(LocalConfiguration.current) { Words(resources) }
    val scope = rememberCoroutineScope()
    var form by remember {
        mutableStateOf(ShareForm(TextFieldValue(draft.title, TextRange(0, draft.title.length)), draft.url))
    }
    var projects by remember { mutableStateOf<List<Project>>(emptyList()) }
    LaunchedEffect(Unit) {
        projects = withContext(Dispatchers.IO) { runCatching { api.projects() }.getOrDefault(emptyList()) }
    }
    // One task only: a title that reads as several tasks is stored as typed.
    val parse = rememberQuickParse(api, form.title.text, enabled = true)?.takeIf { it.drafts.size == 1 && it.readAny() }
    ShareSheet(
        form = form,
        projects = projects,
        read = parse?.let { quickSummary(resources, words, it) },
        today = LocalDate.now(),
        onChange = { form = it },
        onAdd = {
            form = form.copy(adding = true)
            val adding = form
            scope.launch {
                val ok = try {
                    withContext(Dispatchers.IO) { addShared(api, adding, parse, pageTitle) }
                    true
                } catch (e: CancellationException) {
                    throw e
                } catch (e: Exception) {
                    Log.w("adhd-todo", "share failed", e)
                    false
                }
                if (ok) onDone(R.string.share_added) else form = form.copy(adding = false)
            }
        },
        onDismiss = onDismiss,
    )
}

/** Stores the shared task and keeps its link; the link's title is the page's, else the task's. */
private fun addShared(api: CoreApi, form: ShareForm, parse: QuickParse?, pageTitle: String) {
    val title = form.title.text.trim()
    val zone = ZoneId.systemDefault()
    val id = when (val choice = shareWhen(form.picked, parse != null)) {
        ShareWhen.Read -> createQuickDrafts(api, checkNotNull(parse), form.projectId).first()
        ShareWhen.NoDate -> api.createTask(title, form.projectId)
        is ShareWhen.Day -> api.createTask(title, form.projectId, dueAt = dueStamp(choice.option.date, null, zone))
    }
    form.url?.let { api.addLink(id, it, pageTitle.ifEmpty { title }) }
}
