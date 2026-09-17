<template>
  <div>
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>项目列表</span>
          <el-button type="primary" @click="showAddDialog = true">
            <el-icon><Plus /></el-icon> 添加项目
          </el-button>
        </div>
      </template>
      <el-table :data="projects" stripe v-loading="loading">
        <el-table-column prop="gitlab_id" label="GitLab ID" width="110" />
        <el-table-column prop="name" label="项目名称" />
        <el-table-column prop="path" label="路径" />
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'">
              {{ row.enabled ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" align="center">
          <template #default="{ row }">
            <el-button size="small" @click="editProject(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="removeProject(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加/编辑对话框 -->
    <el-dialog v-model="showAddDialog" :title="editingId ? '编辑项目' : '添加项目'" width="520">
      <el-form :model="form" label-width="100">
        <el-form-item label="GitLab ID">
          <div style="display:flex;gap:8px;width:100%">
            <el-input v-model.number="form.gitlab_id" placeholder="输入GitLab项目ID" :disabled="!!editingId" />
            <el-button v-if="!editingId" @click="searchFromGitLab" :loading="searching">从GitLab获取</el-button>
          </div>
        </el-form-item>
        <el-form-item label="项目名称">
          <el-input v-model="form.name" placeholder="项目名称" />
        </el-form-item>
        <el-form-item label="路径">
          <el-input v-model="form.path" placeholder="group/project" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.enabled" active-text="启用" inactive-text="禁用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button type="primary" @click="saveProject" :loading="saving">保存</el-button>
      </template>
    </el-dialog>

    <!-- 从GitLab搜索对话框 -->
    <el-dialog v-model="showSearchDialog" title="从GitLab搜索项目" width="600">
      <el-input v-model="searchKey" placeholder="搜索项目名称" @input="doSearch" clearable>
        <template #append>
          <el-button @click="doSearch" :loading="searching">搜索</el-button>
        </template>
      </el-input>
      <el-table :data="searchResults" style="margin-top:16px" max-height="400" @row-click="selectProject">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="path_with_namespace" label="路径" />
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getProjects, createProject, updateProject, deleteProject, searchGitLabProjects } from '../api'

const projects = ref([])
const loading = ref(false)
const showAddDialog = ref(false)
const showSearchDialog = ref(false)
const editingId = ref(null)
const saving = ref(false)
const searching = ref(false)
const searchKey = ref('')
const searchResults = ref([])

const form = ref({
  gitlab_id: null,
  name: '',
  path: '',
  web_url: '',
  enabled: true,
})

const loadProjects = async () => {
  loading.value = true
  try {
    const { data } = await getProjects()
    projects.value = data.data || []
  } catch (e) {
    ElMessage.error('加载项目列表失败')
  } finally {
    loading.value = false
  }
}

const editProject = (row) => {
  editingId.value = row.id
  form.value = { ...row }
  showAddDialog.value = true
}

const saveProject = async () => {
  if (!form.value.name) {
    ElMessage.warning('请输入项目名称')
    return
  }
  saving.value = true
  try {
    if (editingId.value) {
      await updateProject(editingId.value, {
        name: form.value.name,
        path: form.value.path,
        enabled: form.value.enabled,
      })
      ElMessage.success('更新成功')
    } else {
      if (!form.value.gitlab_id) {
        ElMessage.warning('请输入GitLab ID')
        return
      }
      await createProject(form.value)
      ElMessage.success('添加成功')
    }
    showAddDialog.value = false
    resetForm()
    loadProjects()
  } catch (e) {
    ElMessage.error('保存失败: ' + (e.response?.data?.error || e.message))
  } finally {
    saving.value = false
  }
}

const removeProject = async (id) => {
  try {
    await ElMessageBox.confirm('确定删除该项目？', '确认')
    await deleteProject(id)
    ElMessage.success('删除成功')
    loadProjects()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

const resetForm = () => {
  editingId.value = null
  form.value = { gitlab_id: null, name: '', path: '', web_url: '', enabled: true }
}

const searchFromGitLab = () => {
  showSearchDialog.value = true
}

const doSearch = async () => {
  if (!searchKey.value) return
  searching.value = true
  try {
    const { data } = await searchGitLabProjects(searchKey.value)
    searchResults.value = data.data || []
  } catch (e) {
    ElMessage.error('搜索失败: ' + (e.response?.data?.error || e.message))
  } finally {
    searching.value = false
  }
}

const selectProject = (row) => {
  form.value.gitlab_id = row.id
  form.value.name = row.name
  form.value.path = row.path_with_namespace
  form.value.web_url = row.web_url
  showSearchDialog.value = false
}

onMounted(loadProjects)
</script>

<style scoped>
.card-header { display: flex; align-items: center; justify-content: space-between; }
</style>