<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import { NButton, NSpace } from 'naive-ui'
import DOMPurify from 'dompurify'
import { assetService } from '@/api/services/assets'
import { feedback } from '@/utils/feedback'

const props = withDefaults(
  defineProps<{
    modelValue: string
    placeholder?: string
    minHeight?: number
  }>(),
  { placeholder: '请输入图文详情', minHeight: 280 },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const editor = ref<HTMLDivElement | null>(null)
const imageInput = ref<HTMLInputElement | null>(null)
const uploading = ref(false)

function syncContent(): void {
  const safeHTML = DOMPurify.sanitize(props.modelValue || '', { USE_PROFILES: { html: true } })
  if (editor.value && editor.value.innerHTML !== safeHTML) {
    editor.value.innerHTML = safeHTML
  }
}

function emitContent(): void {
  if (!editor.value) return
  const safeHTML = DOMPurify.sanitize(editor.value.innerHTML, { USE_PROFILES: { html: true } })
  if (editor.value.innerHTML !== safeHTML) {
    editor.value.innerHTML = safeHTML
  }
  emit('update:modelValue', safeHTML)
}

function exec(command: string, value?: string): void {
  editor.value?.focus()
  document.execCommand(command, false, value)
  emitContent()
}

function addLink(): void {
  const url = window.prompt('请输入链接地址（https://）')?.trim()
  if (!url) return
  if (!/^https:\/\//i.test(url)) {
    feedback.message.error('链接地址必须以 https:// 开头')
    return
  }
  exec('createLink', url)
}

function chooseImage(): void {
  imageInput.value?.click()
}

async function uploadImage(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return

  uploading.value = true
  try {
    const result = await assetService.uploadImage('rich_content', file)
    editor.value?.focus()
    document.execCommand('insertImage', false, result.publicUrl)
    emitContent()
  } catch (error) {
    feedback.message.error((error as { message?: string }).message ?? '详情图片上传失败')
  } finally {
    uploading.value = false
  }
}

watch(() => props.modelValue, () => void nextTick(syncContent))
onMounted(syncContent)
</script>

<template>
  <div class="rich-editor">
    <div class="rich-editor__toolbar">
      <NSpace :size="6">
        <NButton
          size="tiny"
          @mousedown.prevent="exec('formatBlock', 'p')"
        >
          正文
        </NButton>
        <NButton
          size="tiny"
          @mousedown.prevent="exec('formatBlock', 'h2')"
        >
          标题
        </NButton>
        <NButton
          size="tiny"
          @mousedown.prevent="exec('bold')"
        >
          加粗
        </NButton>
        <NButton
          size="tiny"
          @mousedown.prevent="exec('italic')"
        >
          斜体
        </NButton>
        <NButton
          size="tiny"
          @mousedown.prevent="exec('insertUnorderedList')"
        >
          无序列表
        </NButton>
        <NButton
          size="tiny"
          @mousedown.prevent="exec('insertOrderedList')"
        >
          有序列表
        </NButton>
        <NButton
          size="tiny"
          @mousedown.prevent="addLink"
        >
          链接
        </NButton>
        <NButton
          size="tiny"
          :loading="uploading"
          @mousedown.prevent="chooseImage"
        >
          插入图片
        </NButton>
        <NButton
          size="tiny"
          @mousedown.prevent="exec('removeFormat')"
        >
          清除格式
        </NButton>
      </NSpace>
      <input
        ref="imageInput"
        class="rich-editor__file"
        type="file"
        accept="image/png,image/jpeg,image/webp"
        @change="uploadImage"
      >
    </div>
    <div
      ref="editor"
      class="rich-editor__content"
      :class="{ 'rich-editor__content--empty': !modelValue }"
      :data-placeholder="placeholder"
      :style="{ minHeight: `${minHeight}px` }"
      contenteditable="true"
      role="textbox"
      aria-multiline="true"
      @input="emitContent"
      @blur="emitContent"
    />
  </div>
</template>

<style scoped>
.rich-editor {
  overflow: hidden;
  width: 100%;
  border: 1px solid var(--ic-color-border-strong);
  border-radius: var(--ic-radius-md);
  background: var(--ic-color-surface);
}

.rich-editor:focus-within { border-color: var(--ic-color-primary); }
.rich-editor__toolbar {
  padding: var(--ic-space-2);
  border-bottom: 1px solid var(--ic-color-border);
  background: var(--ic-color-surface-muted);
}
.rich-editor__file { display: none; }
.rich-editor__content {
  overflow-y: auto;
  max-height: 520px;
  padding: var(--ic-space-4);
  color: var(--ic-color-text);
  line-height: 1.75;
  outline: none;
}
.rich-editor__content--empty::before {
  color: var(--ic-color-text-secondary);
  content: attr(data-placeholder);
  pointer-events: none;
}
.rich-editor__content :deep(h2) { margin: 1.2em 0 0.5em; font-size: var(--ic-font-xl); }
.rich-editor__content :deep(p) { margin: 0.6em 0; }
.rich-editor__content :deep(img) { display: block; max-width: 100%; height: auto; margin: var(--ic-space-4) 0; }
.rich-editor__content :deep(a) { color: var(--ic-color-info); text-decoration: underline; }
</style>
