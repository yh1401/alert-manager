<template>
    <div class="side-by-side">
        <div class="side">
            <div class="diff-title diff-title--from">
                {{ oldTitle }}
            </div>
            <div class="code-lines">
                <div v-for="(line, idx) in leftLines" :key="'l-' + idx" :class="['code-line', lineClass(line.type)]">
                    <span class="line-no">{{ idx + 1 }}</span>
                    <span class="line-text">{{
                        line.text
                    }}</span>
                </div>
            </div>
        </div>
        <div class="side">
            <div class="diff-title diff-title--to">
                {{ newTitle }}
            </div>
            <div class="code-lines">
                <div v-for="(line, idx) in rightLines" :key="'r-' + idx" :class="['code-line', lineClass(line.type)]">
                    <span class="line-no">{{ idx + 1 }}</span>
                    <span class="line-text">{{
                        line.text
                    }}</span>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue';
import * as Diff from 'diff';

const props = defineProps({
    oldTitle: {
        type: String,
        default: 'Old Version',
    },
    newTitle: {
        type: String,
        default: 'New Version',
    },
    oldContent: {
        type: String,
        default: '',
    },
    newContent: {
        type: String,
        default: '',
    },
});

const leftLines = ref([]);
const rightLines = ref([]);

const buildSideBySide = (oldText, newText) => {
    const parts = Diff.diffLines(oldText || "", newText || "");
    const left = [];
    const right = [];

    for (let i = 0; i < parts.length; i++) {
        const part = parts[i];
        const lines = part.value.split("\n");
        if (lines.length && lines[lines.length - 1] === "") lines.pop();

        if (!part.added && !part.removed) {
            lines.forEach((l) => {
                left.push({ text: l, type: "unchanged" });
                right.push({ text: l, type: "unchanged" });
            });
            continue;
        }

        if (part.removed) {
            const next = parts[i + 1];
            if (next && next.added) {
                const oldLines = lines;
                const newLines = next.value.split("\n");
                if (newLines.length && newLines[newLines.length - 1] === "")
                    newLines.pop();
                const maxLen = Math.max(oldLines.length, newLines.length);
                for (let k = 0; k < maxLen; k++) {
                    left.push({ text: oldLines[k] || "", type: "modified" });
                    right.push({ text: newLines[k] || "", type: "modified" });
                }
                i++;
            } else {
                lines.forEach((l) => {
                    left.push({ text: l, type: "deleted" });
                    right.push({ text: "", type: "empty" });
                });
            }
            continue;
        }

        if (part.added) {
            lines.forEach((l) => {
                left.push({ text: "", type: "empty" });
                right.push({ text: l, type: "added" });
            });
            continue;
        }
    }

    leftLines.value = left;
    rightLines.value = right;
};

watch(
    [() => props.oldContent, () => props.newContent],
    ([oldVal, newVal]) => {
        buildSideBySide(oldVal, newVal);
    },
    { immediate: true }
);

const lineClass = (type) => {
    switch (type) {
        case "added":
            return "line-added";
        case "deleted":
            return "line-deleted";
        case "modified":
            return "line-modified";
        default:
            return "";
    }
};
</script>

<style scoped>
.side-by-side {
    display: flex;
    gap: 16px;
}

.side {
    flex: 1;
}

.diff-title {
    padding: 10px 14px;
    border-radius: 8px;
    margin-bottom: 10px;
    font-weight: 600;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.diff-title--from {
    background: linear-gradient(135deg, #fff1f0 0%, #ffccc7 100%);
    color: #cf1322;
}

.diff-title--to {
    background: linear-gradient(135deg, #f6ffed 0%, #d9f7be 100%);
    color: #389e0d;
}

.code-lines {
    border: 1px solid #e8e8e8;
    border-radius: 8px;
    background: #fff;
    max-height: 45vh;
    overflow: auto;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.code-line {
    display: flex;
    gap: 12px;
    padding: 4px 12px;
    font-family:
        ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas,
        "Liberation Mono", "Courier New", monospace;
    transition: background 0.2s ease;
}

.code-line:hover {
    background: #fafafa;
}

.line-no {
    width: 40px;
    color: #999;
    text-align: right;
    font-weight: 500;
}

.line-text {
    flex: 1;
    white-space: pre-wrap;
    word-break: break-word;
}

.line-added {
    background: linear-gradient(90deg,
            rgba(183, 235, 143, 0.15) 0%,
            transparent 100%);
}

.line-deleted {
    background: linear-gradient(90deg,
            rgba(255, 189, 189, 0.15) 0%,
            transparent 100%);
}

.line-modified {
    background: linear-gradient(90deg,
            rgba(145, 213, 255, 0.15) 0%,
            transparent 100%);
}
</style>
