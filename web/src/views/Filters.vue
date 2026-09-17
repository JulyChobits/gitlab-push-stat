<template>
  <div>
    <!-- 数据库过滤规则 -->
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>过滤规则（数据库）</span>
          <el-button type="primary" @click="openAdd">
            <el-icon><Plus /></el-icon> 添加规则
          </el-button>
        </div>
      </template>
      <el-table :data="rules" stripe v-loading="loading">
        <el-table-column prop="rule_type" label="类型" width="120">
          <template #default="{ row }">
            <el-tag :type="typeTagMap[row.rule_type] || 'info'" size="small">
              {{ typeLabel[row.rule_type] || row.rule_type }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="pattern" label="匹配规则" show-overflow-tooltip />
        <el-table-column prop="remark" label="备注" width="200" show-overflow-tooltip />
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" @change="toggleRule(row)" size="small" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" align="center">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="removeRule(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 配置文件过滤规则 -->
    <el-card shadow="hover" style="margin-top: 16px;">
      <template #header>
        <div class="card-header">
          <span>配置文件规则（config.yaml）</span>
          <el-tag type="info" size="small">只读</el-tag>
        </div>
      </template>
      <el-tabs v-model="configActiveTab" v-loading="configLoading">
        <el-tab-pane :label="`排除路径 (${configRules.paths?.length || 0})`" name="paths">
          <el-table :data="configRules.paths || []" stripe size="small" max-height="400">
            <el-table-column type="index" label="#" width="50" />
            <el-table-column prop="" label="路径正则">
              <template #default="{ row }">{{ row }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
        <el-tab-pane :label="`排除扩展名 (${configRules.extensions?.length || 0})`" name="extensions">
          <el-table :data="configRules.extensions || []" stripe size="small" max-height="400">
            <el-table-column type="index" label="#" width="50" />
            <el-table-column prop="" label="扩展名">
              <template #default="{ row }">{{ row }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
        <el-tab-pane :label="`排除文件名 (${configRules.filenames?.length || 0})`" name="filenames">
          <el-table :data="configRules.filenames || []" stripe size="small" max-height="400">
            <el-table-column type="index" label="#" width="50" />
            <el-table-column prop="" label="文件名">
              <template #default="{ row }">{{ row }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
        <el-tab-pane :label="`排除提交信息 (${configRules.messages?.length || 0})`" name="messages">
          <el-table :data="configRules.messages || []" stripe size="small" max-height="400">
            <el-table-column type="index" label="#" width="50" />
            <el-table-column prop="" label="提交信息正则">
              <template #default="{ row }">{{ row }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-dialog v-model="showDialog" title="添加过滤规则" width="520">
      <el-form :model="form" label-width="90">
        <el-form-item label="规则类型">
          <el-select v-model="form.rule_type" style="width:100%">
            <el-option label="文件路径" value="path" />
            <el-option label="文件扩展名" value="extension" />
            <el-option label="文件名" value="filename" />
            <el-option label="提交信息" value="message" />
          </el-select>
        </el-form-item>
        <el-form-item label="匹配模式">
          <el-input v-model="form.pattern" placeholder="正则表达式或精确匹配值" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" placeholder="可选备注" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" @click="saveRule" :loading="saving">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getFilters, createFilter, updateFilter, deleteFilter, getFilterConfigRules } from '../api'

const rules = ref([])
const loading = ref(false)
const showDialog = ref(false)
const saving = ref(false)
const form = ref({ rule_type: 'path', pattern: '', remark: '', enabled: true })

const configRules = ref({ paths: [], extensions: [], filenames: [], messages: [] })
const configLoading = ref(false)
const configActiveTab = ref('paths')

const typeLabel = { path: '文件路径', extension: '扩展名', filename: '文件名', message: '提交信息' }
const typeTagMap = { path: 'primary', extension: 'success', filename: 'warning', message: 'danger' }

const loadRules = async () => {
  loading.value = true
  try {
    const { data } = await getFilters()
    rules.value = data.data || []
  } catch (e) {
    ElMessage.error('加载过滤规则失败')
  } finally {
    loading.value = false
  }
}

const loadConfigRules = async () => {
  configLoading.value = true
  try {
    const { data } = await getFilterConfigRules()
    configRules.value = data.data || { paths: [], extensions: [], filenames: [], messages: [] }
  } catch (e) {
    console.error('加载配置规则失败', e)
  } finally {
    configLoading.value = false
  }
}

const openAdd = () => {
  form.value = { rule_type: 'path', pattern: '', remark: '', enabled: true }
  showDialog.value = true
}

const saveRule = async () => {
  if (!form.value.pattern) { ElMessage.warning('请输入匹配模式'); return }
  saving.value = true
  try {
    await createFilter(form.value)
    ElMessage.success('添加成功')
    showDialog.value = false
    loadRules()
  } catch (e) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const toggleRule = async (row) => {
  try {
    await updateFilter(row.id, { enabled: row.enabled })
  } catch (e) {
    ElMessage.error('更新失败')
    row.enabled = !row.enabled
  }
}

const removeRule = async (id) => {
  try {
    await ElMessageBox.confirm('确定删除该规则？', '确认')
    await deleteFilter(id)
    ElMessage.success('删除成功')
    loadRules()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

onMounted(() => {
  loadRules()
  loadConfigRules()
})
</script>

<style scoped>
.card-header { display: flex; align-items: center; justify-content: space-between; }
</style>