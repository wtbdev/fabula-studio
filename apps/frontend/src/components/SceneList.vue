<script setup lang="ts">
import { Clock, Layers3, MapPin } from 'lucide-vue-next'
import type { SceneDTO } from '../api/types'

const props = defineProps<{
  scenes: SceneDTO[]
  activeSceneId: string | null
  disabled?: boolean
}>()

const emit = defineEmits<{
  select: [sceneId: string]
}>()

const handleSelect = (sceneId: string) => {
  if (props.disabled) return
  emit('select', sceneId)
}
</script>

<template>
  <div class="scene-list-panel" :class="{ 'is-disabled': disabled }">
    <div class="scene-list-header">
      <div>
        <span class="scene-list-label">
          <n-icon><Layers3 /></n-icon>
          剧本场次
        </span>
        <p>按生成顺序梳理剧本结构</p>
      </div>
      <n-tag :bordered="false" size="small" type="success">{{ scenes.length }} 场</n-tag>
    </div>

    <n-scrollbar class="scene-list-scroll">
      <div
        v-for="scene in scenes"
        :key="scene.id"
        class="scene-item"
        :class="{ 'scene-active': scene.id === activeSceneId, 'is-disabled': disabled }"
        :aria-disabled="disabled"
        @click="handleSelect(scene.id)"
      >
        <div class="scene-item-head">
          <span class="scene-no">{{ String(scene.sceneNo).padStart(2, '0') }}</span>
          <span class="scene-title">{{ scene.title }}</span>
        </div>
        <div class="scene-meta">
          <span v-if="scene.location">
            <n-icon :size="12"><MapPin /></n-icon>
            {{ scene.location }}
          </span>
          <span v-if="scene.timeText">
            <n-icon :size="12"><Clock /></n-icon>
            {{ scene.timeText }}
          </span>
        </div>
        <p v-if="scene.summary" class="scene-summary">{{ scene.summary }}</p>
      </div>

      <n-empty
        v-if="scenes.length === 0"
        description="暂无场次"
        class="scene-list-empty"
      />
    </n-scrollbar>
  </div>
</template>

<style scoped>
.scene-list-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--color-line);
  border-radius: 8px;
  background: rgba(255, 253, 248, 0.96);
  box-shadow: var(--shadow-panel);
}

.scene-list-panel.is-disabled {
  background: rgba(255, 253, 248, 0.82);
}

.scene-list-header {
  display: flex;
  position: relative;
  z-index: 1;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px;
  border-bottom: 1px solid var(--color-line);
}

.scene-list-label {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: var(--color-sage);
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 800;
}

.scene-list-header p {
  margin: 6px 0 0;
  color: var(--color-muted);
  font-size: 12px;
  line-height: 1.45;
}

.scene-list-scroll {
  flex: 1;
  min-height: 0;
}

.scene-item {
  position: relative;
  padding: 14px 14px 14px 16px;
  border-bottom: 1px solid var(--color-line);
  cursor: pointer;
  transition:
    background-color 180ms ease,
    border-color 180ms ease;
}

.scene-item.is-disabled {
  cursor: not-allowed;
}

.scene-item:last-child {
  border-bottom: none;
}

.scene-item:not(.is-disabled):hover {
  background: rgba(238, 247, 243, 0.5);
}

.scene-item.scene-active {
  background: var(--color-sage-soft);
}

.scene-item.scene-active::before {
  position: absolute;
  top: 12px;
  bottom: 12px;
  left: 0;
  width: 3px;
  border-radius: 0 999px 999px 0;
  background: var(--color-sage);
  content: "";
}

.scene-item-head {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  min-width: 0;
}

.scene-no {
  display: grid;
  flex-shrink: 0;
  min-width: 28px;
  height: 28px;
  padding: 0 6px;
  place-items: center;
  border: 1px solid rgba(47, 118, 100, 0.18);
  border-radius: 8px;
  color: var(--color-sage);
  background: rgba(238, 247, 243, 0.74);
  font-family: var(--font-display);
  font-size: 12px;
  font-weight: 800;
}

.scene-title {
  min-width: 0;
  color: var(--color-ink);
  font-size: 14px;
  font-weight: 600;
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.scene-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 9px;
  color: var(--color-muted);
  font-size: 12px;
}

.scene-meta span {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.scene-summary {
  margin: 8px 0 0;
  color: var(--color-muted);
  font-size: 13px;
  line-height: 1.6;
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.scene-list-empty {
  padding: 32px 0;
}
</style>
