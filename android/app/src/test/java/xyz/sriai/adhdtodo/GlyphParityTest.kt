package xyz.sriai.adhdtodo

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import xyz.sriai.adhdtodo.ui.theme.Glyphs
import java.io.File

/**
 * Verification 2 of A2a: the Kotlin glyph grids equal the Go grids. The test parses the literals in
 * app/theme/glyph.go, rebuilds each status with the same overlays and compares every cell.
 */
class GlyphParityTest {

    @Test
    fun `kotlin glyphs equal go glyphs`() {
        val source = repoFile("app/theme/glyph.go").readText()
        val disc = goGrid(source, "glyphDisc")
        val ring = goGrid(source, "glyphRing")
        assertEquals(12, disc.size)
        assertEquals(12, ring.size)
        assertTrue(disc.all { it.length == 12 })
        assertTrue(ring.all { it.length == 12 })
        assertEquals(disc, Glyphs.disc)
        assertEquals(ring, Glyphs.ring)

        val doing = goCells(source, "glyphDoingHalf")
        val review = goCells(source, "glyphReviewDot")
        val check = goCells(source, "glyphCheck")
        val bang = goCells(source, "glyphBang")
        val slash = goCells(source, "glyphCancelSlash")
        assertTrue(doing.isNotEmpty() && review.isNotEmpty() && check.isNotEmpty() && bang.isNotEmpty() && slash.isNotEmpty())

        val dashed = HashSet<Pair<Int, Int>>()
        var i = 0
        for (y in ring.indices) {
            for (x in ring[y].indices) {
                if (ring[y][x] == '#') {
                    if (i % 3 == 2) dashed.add(x to y)
                    i++
                }
            }
        }

        val cases = mapOf(
            "backlog" to expected(ring, emptyList(), 'c', dashed),
            "todo" to expected(ring, emptyList(), 'c'),
            "doing" to expected(ring, doing, 'c'),
            "review" to expected(ring, review, 'c'),
            "blocked" to expected(disc, bang, 'w'),
            "done" to expected(disc, check, 'w'),
            "cancelled" to expected(ring, slash, 'c'),
        )
        for ((status, want) in cases) {
            assertEquals("glyph $status", want, Glyphs.rows(status))
        }
    }

    private fun expected(
        base: List<String>,
        overlay: List<Pair<Int, Int>>,
        mark: Char,
        drop: Set<Pair<Int, Int>> = emptySet(),
    ): List<String> {
        val cells = Array(12) { CharArray(12) { '.' } }
        for (y in 0 until 12) {
            for (x in 0 until 12) {
                cells[y][x] = when {
                    base[y][x] != '#' -> base[y][x]
                    (x to y) in drop -> '.'
                    else -> 'c'
                }
            }
        }
        for ((x, y) in overlay) cells[y][x] = mark
        return cells.map { String(it) }
    }

    private fun goGrid(source: String, name: String): List<String> {
        val block = Regex("""$name\s*=\s*\[\]string\{(.*?)\}""", RegexOption.DOT_MATCHES_ALL)
            .find(source)?.groupValues?.get(1)
            ?: error("$name not found in glyph.go")
        return Regex("\"([.#]{12})\"").findAll(block).map { it.groupValues[1] }.toList()
    }

    private fun goCells(source: String, name: String): List<Pair<Int, Int>> {
        val line = source.lineSequence().firstOrNull { it.contains(name) }
            ?: error("$name not found in glyph.go")
        return Regex("""\{(\d+),\s*(\d+)\}""").findAll(line).map {
            it.groupValues[1].toInt() to it.groupValues[2].toInt()
        }.toList()
    }

    private fun repoFile(path: String): File {
        var dir: File? = File("").absoluteFile
        while (dir != null) {
            val candidate = File(dir, path)
            if (candidate.exists()) return candidate
            dir = dir.parentFile
        }
        error("could not find $path above ${File("").absolutePath}")
    }
}
