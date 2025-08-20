// 全局变量
let currentPath = '.';
let selectedFiles = [];
let selectedScripts = [];
let currentSection = 'monitoring';
let currentLanguage = 'zh-CN';

// 多语言支持
const translations = {
    'zh-CN': {
        // 通用
        'server_status': '服务器状态',
        'server_management_system': '服务器管理系统',
        'running': '运行中',
        'refresh': '刷新',
        'cancel': '取消',
        'close': '关闭',
        'delete': '删除',
        'create': '创建',
        'rename': '重命名',
        'success': '成功',
        'error': '错误',
        'confirm': '确定',
        'loading': '加载中',
        
        // 导航
        'directory_management': '目录管理',
        'docker_scripts': 'Docker脚本',
        'container_management': '容器管理',
        'image_management': '镜像管理',
        'port_usage': '端口占用',
        'monitoring': '系统监控',
        'service_management': '服务管理',
        'system_monitoring': '系统监控',
        'start_monitoring': '开始监控',
        'stop_monitoring': '停止监控',
        'monitoring_status': '监控状态',
        'cpu_usage': 'CPU 使用率',
        'memory_usage': '内存使用',
        'disk_usage': '磁盘使用',
        'system_uptime': '系统运行时间',
        'last_update': '最后更新时间',
        'connected': '已连接',
        'disconnected': '未连接',
        'connecting': '连接中...',
        'connection_error': '连接错误',
        
        // 目录管理
        'parent_directory': '上级目录',
        'new_directory': '新建目录',
        'delete_selected': '删除选中',
        'rename_file': '重命名',
        'directory_name': '目录名称',
        'enter_directory_name': '请输入目录名称',
        'directory_created': '目录创建成功',
        'directory_create_failed': '创建目录失败',
        'select_file_to_delete': '请选择要删除的文件或目录',
        'delete_confirm': '确定要删除选中的 {count} 个项目吗？',
        'delete_success': '删除成功',
        'delete_failed': '删除失败',
        'select_one_file': '请选择一个文件或目录进行重命名',
        'new_name': '新名称',
        'enter_new_name': '请输入新名称',
        'rename_success': '重命名成功',
        'rename_failed': '重命名失败',
        'directory_refreshed': '目录已刷新',
        'name': '名称',
        'type': '类型',
        'size': '大小',
        'modified_time': '修改时间',
        'directory': '目录',
        'file': '文件',
        
        // Docker脚本
        'create_script': '创建脚本',
        'delete_script': '删除脚本',
        'script_name': '脚本名称',
        'script_content': '脚本内容',
        'enter_script_name': '请输入脚本名称',
        'enter_script_content': '请填写脚本名称和内容',
        'script_created': '脚本创建成功',
        'script_create_failed': '创建脚本失败',
        'view_script': '查看',
        'execute_script': '执行',
        'logs': '日志',
        'script_execution_confirm': '确定要执行此脚本吗？',
        'script_execution_success': '脚本执行成功',
        'script_execution_failed': '脚本执行失败',
        'execution_result': '脚本执行结果',
        'execution_status': '执行状态',
        'execution_time': '执行时间',
        'execution_duration': '执行耗时',
        'execution_output': '执行结果',
        'select_script_to_delete': '请选择要删除的脚本',
        'delete_script_confirm': '确定要删除选中的 {count} 个脚本吗？',
        'script_deleted': '删除成功',
        'script_delete_failed': '删除失败',
        'no_execution_logs': '该脚本暂无执行日志',
        'execution_logs_failed': '获取执行日志失败',
        'script_path': '路径',
        'script_list_refreshed': '脚本列表已刷新',
        'no_logs': '暂无执行日志',
        'logs_refreshed': '日志已刷新',
        
        // 容器管理
        'container_id': '容器ID',
        'container_name': '名称',
        'container_image': '镜像',
        'container_status': '状态',
        'container_ports': '端口',
        'container_created': '创建时间',
        'container_actions': '操作',
        'start': '启动',
        'stop': '停止',
        'restart': '重启',
        'container_logs': '日志',
        'remove': '删除',
        'container_start_success': '容器启动成功',
        'container_start_failed': '启动容器失败',
        'container_stop_confirm': '确定要停止此容器吗？',
        'container_stop_success': '容器停止成功',
        'container_stop_failed': '停止容器失败',
        'container_restart_success': '容器重启成功',
        'container_restart_failed': '重启容器失败',
        'container_remove_confirm': '确定要删除此容器吗？此操作不可恢复！',
        'container_remove_success': '容器删除成功',
        'container_remove_failed': '删除容器失败',
        'container_logs_failed': '获取容器日志失败',
        'container_list_refreshed': '容器列表已刷新',
        
        // 镜像管理
        'image_id': '镜像ID',
        'image_repository': '仓库',
        'image_tag': '标签',
        'image_size': '大小',
        'image_created': '创建时间',
        'pull_image': '拉取镜像',
        'prune_images': '清理镜像',
        'image_name': '镜像名称',
        'enter_image_name': '请输入镜像名称',
        'image_pull_success': '镜像拉取成功',
        'image_pull_failed': '拉取镜像失败',
        'pull_result': '镜像拉取结果',
        'image_remove_confirm': '确定要删除此镜像吗？此操作不可恢复！',
        'image_remove_success': '镜像删除成功',
        'image_remove_failed': '删除镜像失败',
        'prune_images_confirm': '确定要清理所有未使用的镜像吗？此操作不可恢复！',
        'prune_result': '镜像清理结果',
        'image_list_refreshed': '镜像列表已刷新',
        
        // 端口管理
        'port': '端口',
        'process': '进程',
        'pid': 'PID',
        'status': '状态',
        'listening': '监听中',
        'not_listening': '未监听',
        'unknown': '未知',
        'port_info_refreshed': '端口信息已刷新',
        'port_info_failed': '加载端口信息失败',
        
        // 服务管理
        'service_name': '服务名称',
        'service_status': '状态',
        'memory': '内存',
        'cpu': 'CPU',
        'service_start_success': '服务启动成功',
        'service_start_failed': '启动服务失败',
        'service_stop_success': '服务停止成功',
        'service_stop_failed': '停止服务失败',
        'service_restart_success': '服务重启成功',
        'service_restart_failed': '重启服务失败',
        'service_info_refreshed': '服务信息已刷新',
        'service_info_failed': '加载服务信息失败',
        
        // 通知
        'load_directory_failed': '加载目录失败',
        'load_scripts_failed': '加载脚本失败',
        'load_containers_failed': '加载容器列表失败',
        'load_images_failed': '加载镜像列表失败',
        'load_services_failed': '加载服务信息失败'
    },
    'en-US': {
        // 通用
        'server_status': 'Server Status',
        'server_management_system': 'Server Management System',
        'running': 'Running',
        'refresh': 'Refresh',
        'cancel': 'Cancel',
        'close': 'Close',
        'delete': 'Delete',
        'create': 'Create',
        'rename': 'Rename',
        'success': 'Success',
        'error': 'Error',
        'confirm': 'Confirm',
        'loading': 'Loading',
        
        // 导航
        'directory_management': 'Directory Management',
        'docker_scripts': 'Docker Scripts',
        'container_management': 'Container Management',
        'image_management': 'Image Management',
        'port_usage': 'Port Usage',
        'service_management': 'Service Management',
        'monitoring': 'System Monitoring',
        'system_monitoring': 'System Monitoring',
        'start_monitoring': 'Start Monitoring',
        'stop_monitoring': 'Stop Monitoring',
        'monitoring_status': 'Monitoring Status',
        'cpu_usage': 'CPU Usage',
        'memory_usage': 'Memory Usage',
        'disk_usage': 'Disk Usage',
        'system_uptime': 'System Uptime',
        'last_update': 'Last Update',
        'connected': 'Connected',
        'disconnected': 'Disconnected',
        'connecting': 'Connecting...',
        'connection_error': 'Connection Error',
        
        // 目录管理
        'parent_directory': 'Parent Directory',
        'new_directory': 'New Directory',
        'delete_selected': 'Delete Selected',
        'rename_file': 'Rename',
        'directory_name': 'Directory Name',
        'enter_directory_name': 'Please enter directory name',
        'directory_created': 'Directory created successfully',
        'directory_create_failed': 'Failed to create directory',
        'select_file_to_delete': 'Please select files or directories to delete',
        'delete_confirm': 'Are you sure you want to delete {count} selected items?',
        'delete_success': 'Delete successful',
        'delete_failed': 'Delete failed',
        'select_one_file': 'Please select one file or directory to rename',
        'new_name': 'New Name',
        'enter_new_name': 'Please enter new name',
        'rename_success': 'Rename successful',
        'rename_failed': 'Rename failed',
        'directory_refreshed': 'Directory refreshed',
        'name': 'Name',
        'type': 'Type',
        'size': 'Size',
        'modified_time': 'Modified Time',
        'directory': 'Directory',
        'file': 'File',
        
        // Docker脚本
        'create_script': 'Create Script',
        'delete_script': 'Delete Script',
        'script_name': 'Script Name',
        'script_content': 'Script Content',
        'enter_script_name': 'Please enter script name',
        'enter_script_content': 'Please fill in script name and content',
        'script_created': 'Script created successfully',
        'script_create_failed': 'Failed to create script',
        'view_script': 'View',
        'execute_script': 'Execute',
        'logs': 'Logs',
        'script_execution_confirm': 'Are you sure you want to execute this script?',
        'script_execution_success': 'Script executed successfully',
        'script_execution_failed': 'Script execution failed',
        'execution_result': 'Script Execution Result',
        'execution_status': 'Execution Status',
        'execution_time': 'Execution Time',
        'execution_duration': 'Execution Duration',
        'execution_output': 'Execution Output',
        'select_script_to_delete': 'Please select scripts to delete',
        'delete_script_confirm': 'Are you sure you want to delete {count} selected scripts?',
        'script_deleted': 'Delete successful',
        'script_delete_failed': 'Delete failed',
        'no_execution_logs': 'No execution logs for this script',
        'execution_logs_failed': 'Failed to get execution logs',
        'script_path': 'Path',
        'script_list_refreshed': 'Script list refreshed',
        'no_logs': 'No execution logs',
        'logs_refreshed': 'Logs refreshed',
        
        // 容器管理
        'container_id': 'Container ID',
        'container_name': 'Name',
        'container_image': 'Image',
        'container_status': 'Status',
        'container_ports': 'Ports',
        'container_created': 'Created',
        'container_actions': 'Actions',
        'start': 'Start',
        'stop': 'Stop',
        'restart': 'Restart',
        'container_logs': 'Logs',
        'remove': 'Remove',
        'container_start_success': 'Container started successfully',
        'container_start_failed': 'Failed to start container',
        'container_stop_confirm': 'Are you sure you want to stop this container?',
        'container_stop_success': 'Container stopped successfully',
        'container_stop_failed': 'Failed to stop container',
        'container_restart_success': 'Container restarted successfully',
        'container_restart_failed': 'Failed to restart container',
        'container_remove_confirm': 'Are you sure you want to remove this container? This operation cannot be undone!',
        'container_remove_success': 'Container removed successfully',
        'container_remove_failed': 'Failed to remove container',
        'container_logs_failed': 'Failed to get container logs',
        'container_list_refreshed': 'Container list refreshed',
        
        // 镜像管理
        'image_id': 'Image ID',
        'image_repository': 'Repository',
        'image_tag': 'Tag',
        'image_size': 'Size',
        'image_created': 'Created',
        'pull_image': 'Pull Image',
        'prune_images': 'Prune Images',
        'image_name': 'Image Name',
        'enter_image_name': 'Please enter image name',
        'image_pull_success': 'Image pulled successfully',
        'image_pull_failed': 'Failed to pull image',
        'pull_result': 'Image Pull Result',
        'image_remove_confirm': 'Are you sure you want to remove this image? This operation cannot be undone!',
        'image_remove_success': 'Image removed successfully',
        'image_remove_failed': 'Failed to remove image',
        'prune_images_confirm': 'Are you sure you want to prune all unused images? This operation cannot be undone!',
        'prune_result': 'Image Prune Result',
        'image_list_refreshed': 'Image list refreshed',
        
        // 端口管理
        'port': 'Port',
        'process': 'Process',
        'pid': 'PID',
        'status': 'Status',
        'listening': 'Listening',
        'not_listening': 'Not Listening',
        'unknown': 'Unknown',
        'port_info_refreshed': 'Port information refreshed',
        'port_info_failed': 'Failed to load port information',
        
        // 服务管理
        'service_name': 'Service Name',
        'service_status': 'Status',
        'memory': 'Memory',
        'cpu': 'CPU',
        'service_start_success': 'Service started successfully',
        'service_start_failed': 'Failed to start service',
        'service_stop_success': 'Service stopped successfully',
        'service_stop_failed': 'Failed to stop service',
        'service_restart_success': 'Service restarted successfully',
        'service_restart_failed': 'Failed to restart service',
        'service_info_refreshed': 'Service information refreshed',
        'service_info_failed': 'Failed to load service information',
        
        // 通知
        'load_directory_failed': 'Failed to load directory',
        'load_scripts_failed': 'Failed to load scripts',
        'load_containers_failed': 'Failed to load container list',
        'load_images_failed': 'Failed to load image list',
        'load_services_failed': 'Failed to load service information'
    }
};

// 获取翻译文本
function t(key) {
    return translations[currentLanguage][key] || key;
}

// 切换语言
function switchLanguage(lang) {
    currentLanguage = lang;
    localStorage.setItem('preferredLanguage', lang);
    updateUILanguage();
}

// 更新界面语言
function updateUILanguage() {
    // 更新HTML语言属性
    document.documentElement.lang = currentLanguage;
    
    // 更新标题
    document.title = t('server_management_system');
    
    // 更新主要标题
    document.querySelector('header h1').textContent = t('server_management_system');
    document.querySelector('#server-status').innerHTML = `${t('server_status')}: <span class="status-running">${t('running')}</span>`;
    
    // 更新语言选择器
    const languageSelect = document.getElementById('language-select');
    if (languageSelect) {
        languageSelect.value = currentLanguage;
    }
    
    // 更新导航按钮
    const navButtons = document.querySelectorAll('.nav-btn');
    navButtons[0].textContent = t('monitoring');
    navButtons[1].textContent = t('directory_management');
    navButtons[2].textContent = t('docker_scripts');
    navButtons[3].textContent = t('container_management');
    navButtons[4].textContent = t('image_management');
    navButtons[5].textContent = t('port_usage');
    navButtons[6].textContent = t('service_management');
    
    // 更新当前区域的语言
    if (currentSection) {
        updateSectionLanguage(currentSection);
    }
}

// 更新特定区域的语言
function updateSectionLanguage(section) {
    switch(section) {
        case 'directory':
            updateDirectoryLanguage();
            break;
        case 'docker':
            updateDockerLanguage();
            break;
        case 'containers':
            updateContainersLanguage();
            break;
        case 'images':
            updateImagesLanguage();
            break;
        case 'ports':
            updatePortsLanguage();
            break;
        case 'services':
            updateServicesLanguage();
            break;
        case 'monitoring':
            updateMonitoringLanguage();
            break;
    }
}

// 更新目录管理区域语言
function updateDirectoryLanguage() {
    const section = document.querySelector('#directory-section');
    if (!section) return;
    
    section.querySelector('h2').textContent = t('directory_management');
    const buttons = section.querySelectorAll('.action-bar button');
    buttons[0].textContent = t('new_directory');
    buttons[1].textContent = t('delete_selected');
    buttons[2].textContent = t('rename_file');
    
    const pathButtons = section.querySelectorAll('.path-navigator button');
    pathButtons[0].textContent = t('parent_directory');
    pathButtons[1].textContent = t('refresh');
    
    // 更新表头
    const headers = section.querySelectorAll('.file-list-header span');
    headers[0].textContent = t('name');
    headers[1].textContent = t('type');
    headers[2].textContent = t('size');
    headers[3].textContent = t('modified_time');
}

// 更新Docker脚本区域语言
function updateDockerLanguage() {
    const section = document.querySelector('#docker-section');
    if (!section) return;
    
    section.querySelector('h2').textContent = t('docker_scripts');
    const buttons = section.querySelectorAll('.action-bar button');
    buttons[0].textContent = t('create_script');
    buttons[1].textContent = t('delete_script');
    
    const refreshBtn = section.querySelector('.path-navigator button');
    if (refreshBtn) refreshBtn.textContent = t('refresh');
    
    // 更新表头
    const headers = section.querySelectorAll('.script-list-header span');
    headers[0].textContent = t('script_name');
    headers[1].textContent = t('script_path');
    headers[2].textContent = t('logs');
}

// 更新容器管理区域语言
function updateContainersLanguage() {
    const section = document.querySelector('#containers-section');
    if (!section) return;
    
    section.querySelector('h2').textContent = t('container_management');
    const refreshBtn = section.querySelector('.section-header button');
    if (refreshBtn) refreshBtn.textContent = t('refresh');
    
    // 更新表头
    const headers = section.querySelectorAll('.container-list-header span');
    headers[0].textContent = t('container_id');
    headers[1].textContent = t('container_name');
    headers[2].textContent = t('container_image');
    headers[3].textContent = t('container_status');
    headers[4].textContent = t('container_ports');
    headers[5].textContent = t('container_created');
    headers[6].textContent = t('container_actions');
}

// 更新镜像管理区域语言
function updateImagesLanguage() {
    const section = document.querySelector('#images-section');
    if (!section) return;
    
    section.querySelector('h2').textContent = t('image_management');
    const buttons = section.querySelectorAll('.action-bar button');
    buttons[0].textContent = t('pull_image');
    buttons[1].textContent = t('prune_images');
    buttons[2].textContent = t('refresh');
    
    // 更新表头
    const headers = section.querySelectorAll('.image-list-header span');
    headers[0].textContent = t('image_id');
    headers[1].textContent = t('image_repository');
    headers[2].textContent = t('image_tag');
    headers[3].textContent = t('image_size');
    headers[4].textContent = t('image_created');
    headers[5].textContent = t('container_actions');
}

// 更新端口管理区域语言
function updatePortsLanguage() {
    const section = document.querySelector('#ports-section');
    if (!section) return;
    
    section.querySelector('h2').textContent = t('port_usage');
    const refreshBtn = section.querySelector('.section-header button');
    if (refreshBtn) refreshBtn.textContent = t('refresh');
    
    // 更新表头
    const headers = section.querySelectorAll('.port-list-header span');
    headers[0].textContent = t('port');
    headers[1].textContent = t('process');
    headers[2].textContent = t('pid');
    headers[3].textContent = t('status');
}

// 更新服务管理区域语言
function updateServicesLanguage() {
    const section = document.querySelector('#services-section');
    if (!section) return;
    
    section.querySelector('h2').textContent = t('service_management');
    const refreshBtn = section.querySelector('.section-header button');
    if (refreshBtn) refreshBtn.textContent = t('refresh');
    
    // 更新表头
    const headers = section.querySelectorAll('.service-list-header span');
    headers[0].textContent = t('service_name');
    headers[1].textContent = t('service_status');
    headers[2].textContent = t('pid');
    headers[3].textContent = t('memory');
    headers[4].textContent = t('cpu');
    headers[5].textContent = t('container_actions');
}

// 更新系统监控区域语言
function updateMonitoringLanguage() {
    const section = document.querySelector('#monitoring-section');
    if (!section) return;
    
    section.querySelector('h2').textContent = t('system_monitoring');
    const toggleBtn = document.getElementById('monitoring-toggle-btn');
    if (toggleBtn) {
        toggleBtn.textContent = isMonitoring ? t('stop_monitoring') : t('start_monitoring');
    }
    
    // 更新监控卡片标题
    const cards = section.querySelectorAll('.monitoring-card h3');
    if (cards.length >= 3) {
        cards[0].textContent = t('cpu_usage');
        cards[1].textContent = t('memory_usage');
        cards[2].textContent = t('disk_usage');
    }
    
    // 更新信息卡片标题
    const infoCards = section.querySelectorAll('.info-card h3');
    if (infoCards.length >= 2) {
        infoCards[0].textContent = t('system_uptime');
        infoCards[1].textContent = t('last_update');
    }
}

// 初始化
document.addEventListener('DOMContentLoaded', function() {
    // 加载保存的语言设置
    const savedLanguage = localStorage.getItem('preferredLanguage');
    if (savedLanguage && translations[savedLanguage]) {
        currentLanguage = savedLanguage;
    }
    
    updateTime();
    setInterval(updateTime, 1000);
    loadDirectory();
    setupEventListeners();
    updateUILanguage();
    
    // 初始化图表，但不立即开始绘制
    initCharts();
});

// 更新时间
function updateTime() {
    const now = new Date();
    document.getElementById('current-time').textContent = now.toLocaleString(currentLanguage);
}

// 设置事件监听器
function setupEventListeners() {
    // 全选复选框
    document.getElementById('select-all').addEventListener('change', function() {
        const checkboxes = document.querySelectorAll('#file-list-content input[type="checkbox"]');
        checkboxes.forEach(cb => cb.checked = this.checked);
        updateSelectedFiles();
    });
    
    document.getElementById('select-all-scripts').addEventListener('change', function() {
        const checkboxes = document.querySelectorAll('#script-list-content input[type="checkbox"]');
        checkboxes.forEach(cb => cb.checked = this.checked);
        updateSelectedScripts();
    });
}

// 显示不同的部分
function showSection(section) {
    // 更新导航按钮
    document.querySelectorAll('.nav-btn').forEach(btn => btn.classList.remove('active'));
    event.target.classList.add('active');
    
    // 更新内容区域
    document.querySelectorAll('.content-section').forEach(sec => sec.classList.remove('active'));
    document.getElementById(section + '-section').classList.add('active');
    
    currentSection = section;
    
    // 更新该部分的语言
    updateSectionLanguage(section);
    
    // 加载相应数据
    switch(section) {
        case 'directory':
            loadDirectory();
            break;
        case 'docker':
            loadDockerScripts();
            break;
        case 'ports':
            loadPorts();
            break;
        case 'services':
            loadServices();
            break;
        case 'containers':
            loadContainers();
            break;
        case 'images':
            loadImages();
            break;
        case 'monitoring':
            // 监控页面不需要加载数据，用户需要手动点击开始监控
            break;
    }
}

// 目录管理功能
function loadDirectory() {
    fetch(`/api/directory?path=${encodeURIComponent(currentPath)}`)
        .then(response => response.json())
        .then(data => {
            displayFiles(data);
        })
        .catch(error => {
            showNotification(t('load_directory_failed') + ': ' + error.message, 'error');
        });
}

function displayFiles(files) {
    const container = document.getElementById('file-list-content');
    container.innerHTML = '';
    
    files.forEach(file => {
        const fileItem = document.createElement('div');
        fileItem.className = 'file-item';
        
        const checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.value = file.path;
        checkbox.addEventListener('change', updateSelectedFiles);
        
        const name = document.createElement('span');
        name.className = 'file-name';
        name.textContent = file.name;
        if (file.isDir) {
            name.style.cursor = 'pointer';
            name.addEventListener('click', () => navigateToDirectory(file.path));
        }
        
        const type = document.createElement('span');
        type.className = 'file-type';
        type.textContent = file.isDir ? t('directory') : t('file');
        
        const size = document.createElement('span');
        size.textContent = file.isDir ? '-' : formatFileSize(file.size);
        
        const modTime = document.createElement('span');
        modTime.textContent = file.modTime;
        
        fileItem.appendChild(checkbox);
        fileItem.appendChild(name);
        fileItem.appendChild(type);
        fileItem.appendChild(size);
        fileItem.appendChild(modTime);
        
        container.appendChild(fileItem);
    });
    
    // 更新表头语言
    updateDirectoryLanguage();
}

function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function navigateToDirectory(path) {
    currentPath = path;
    document.getElementById('current-path').value = path;
    loadDirectory();
}

function goToParent() {
    if (currentPath === '.') return;
    
    const parentPath = currentPath.split('/').slice(0, -1).join('/');
    currentPath = parentPath || '.';
    document.getElementById('current-path').value = currentPath;
    loadDirectory();
}

function refreshDirectory() {
    loadDirectory();
    showNotification(t('directory_refreshed'), 'success');
}

function updateSelectedFiles() {
    const checkboxes = document.querySelectorAll('#file-list-content input[type="checkbox"]:checked');
    selectedFiles = Array.from(checkboxes).map(cb => cb.value);
    
    // 更新全选复选框状态
    const allCheckboxes = document.querySelectorAll('#file-list-content input[type="checkbox"]');
    const selectAllCheckbox = document.getElementById('select-all');
    selectAllCheckbox.checked = allCheckboxes.length > 0 && checkboxes.length === allCheckboxes.length;
}

function showCreateDirDialog() {
    const content = `
        <div class="form-group">
            <label for="dir-name">${t('directory_name')}:</label>
            <input type="text" id="dir-name" placeholder="${t('enter_directory_name')}">
        </div>
        <div class="form-actions">
            <button onclick="createDirectory()" class="btn-primary">${t('create')}</button>
            <button onclick="closeDialog()" class="btn-secondary">${t('cancel')}</button>
        </div>
    `;
    showDialog(t('new_directory'), content);
}

function createDirectory() {
    const dirName = document.getElementById('dir-name').value.trim();
    if (!dirName) {
        showNotification(t('enter_directory_name'), 'error');
        return;
    }
    
    fetch('/api/directory', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            path: currentPath,
            name: dirName
        })
    })
    .then(response => {
        if (response.ok) {
            showNotification(t('directory_created'), 'success');
            closeDialog();
            loadDirectory();
        } else {
            throw new Error(t('create') + t('error'));
        }
    })
    .catch(error => {
        showNotification(t('directory_create_failed') + ': ' + error.message, 'error');
    });
}

function deleteSelected() {
    if (selectedFiles.length === 0) {
        showNotification('请选择要删除的文件或目录', 'error');
        return;
    }
    
    if (!confirm(`确定要删除选中的 ${selectedFiles.length} 个项目吗？`)) {
        return;
    }
    
    selectedFiles.forEach(path => {
        fetch('/api/directory', {
            method: 'DELETE',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ path: path })
        })
        .then(response => {
            if (response.ok) {
                showNotification('删除成功', 'success');
            } else {
                throw new Error('删除失败');
            }
        })
        .catch(error => {
            showNotification('删除失败: ' + error.message, 'error');
        });
    });
    
    setTimeout(() => {
        loadDirectory();
        selectedFiles = [];
    }, 1000);
}

function showRenameDialog() {
    if (selectedFiles.length !== 1) {
        showNotification('请选择一个文件或目录进行重命名', 'error');
        return;
    }
    
    const oldPath = selectedFiles[0];
    const oldName = oldPath.split('/').pop();
    
    const content = `
        <div class="form-group">
            <label for="new-name">新名称:</label>
            <input type="text" id="new-name" value="${oldName}" placeholder="请输入新名称">
        </div>
        <div class="form-actions">
            <button onclick="renameDirectory('${oldPath}')" class="btn-primary">重命名</button>
            <button onclick="closeDialog()" class="btn-secondary">取消</button>
        </div>
    `;
    showDialog('重命名', content);
}

function renameDirectory(oldPath) {
    const newName = document.getElementById('new-name').value.trim();
    if (!newName) {
        showNotification('请输入新名称', 'error');
        return;
    }
    
    const parentPath = oldPath.split('/').slice(0, -1).join('/');
    const newPath = parentPath + '/' + newName;
    
    fetch('/api/directory/rename', {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            oldPath: oldPath,
            newPath: newPath
        })
    })
    .then(response => {
        if (response.ok) {
            showNotification('重命名成功', 'success');
            closeDialog();
            loadDirectory();
        } else {
            throw new Error('重命名失败');
        }
    })
    .catch(error => {
        showNotification('重命名失败: ' + error.message, 'error');
    });
}

// Docker脚本管理功能
function loadDockerScripts() {
    fetch(`/api/docker-scripts?path=${encodeURIComponent(currentPath)}`)
        .then(response => response.json())
        .then(data => {
            // 确保data是数组，如果不是则使用空数组
            const scripts = Array.isArray(data) ? data : [];
            displayScripts(scripts);
        })
        .catch(error => {
            showNotification('加载脚本失败: ' + error.message, 'error');
        });
}

function displayScripts(scripts) {
    const container = document.getElementById('script-list-content');
    container.innerHTML = '';
    
    // 确保scripts是数组
    if (!Array.isArray(scripts)) {
        scripts = [];
    }
    
    scripts.forEach(script => {
        const scriptItem = document.createElement('div');
        scriptItem.className = 'script-item';
        
        const checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.value = script.path;
        checkbox.addEventListener('change', updateSelectedScripts);
        
        const name = document.createElement('span');
        name.textContent = script.name;
        
        const path = document.createElement('span');
        path.textContent = script.path;
        
        const actions = document.createElement('span');
        actions.innerHTML = `
            <button onclick="viewScript('${script.path}')" class="btn-execute">查看</button>
            <button onclick="executeScript('${script.path}')" class="btn-execute">执行</button>
            <button onclick="viewExecutionLogs(${script.id || 0}, '${script.name}')" class="btn-execute">日志</button>
        `;
        
        scriptItem.appendChild(checkbox);
        scriptItem.appendChild(name);
        scriptItem.appendChild(path);
        scriptItem.appendChild(actions);
        
        container.appendChild(scriptItem);
    });
}

function updateSelectedScripts() {
    const checkboxes = document.querySelectorAll('#script-list-content input[type="checkbox"]:checked');
    selectedScripts = Array.from(checkboxes).map(cb => cb.value);
    
    // 更新全选复选框状态
    const allCheckboxes = document.querySelectorAll('#script-list-content input[type="checkbox"]');
    const selectAllCheckbox = document.getElementById('select-all-scripts');
    selectAllCheckbox.checked = allCheckboxes.length > 0 && checkboxes.length === allCheckboxes.length;
}

function refreshDockerScripts() {
    loadDockerScripts();
    showNotification('脚本列表已刷新', 'success');
}

function showCreateScriptDialog() {
    const content = `
        <div class="form-group">
            <label for="script-name">脚本名称:</label>
            <input type="text" id="script-name" placeholder="请输入脚本名称">
        </div>
        <div class="form-group">
            <label for="script-content">脚本内容:</label>
            <textarea id="script-content" placeholder="#!/bin/bash&#10;# Docker启动脚本"></textarea>
        </div>
        <div class="form-actions">
            <button onclick="createScript()" class="btn-primary">创建</button>
            <button onclick="closeDialog()" class="btn-secondary">取消</button>
        </div>
    `;
    showDialog('创建Docker脚本', content);
}

function createScript() {
    const scriptName = document.getElementById('script-name').value.trim();
    const scriptContent = document.getElementById('script-content').value.trim();
    
    if (!scriptName || !scriptContent) {
        showNotification('请填写脚本名称和内容', 'error');
        return;
    }
    
    fetch('/api/docker-scripts', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            path: currentPath,
            name: scriptName,
            content: scriptContent
        })
    })
    .then(response => {
        if (response.ok) {
            showNotification('脚本创建成功', 'success');
            closeDialog();
            loadDockerScripts();
        } else {
            throw new Error('创建失败');
        }
    })
    .catch(error => {
        showNotification('创建脚本失败: ' + error.message, 'error');
    });
}

function viewScript(path) {
    fetch(`/api/docker-scripts?path=${encodeURIComponent(path)}`)
        .then(response => response.json())
        .then(data => {
            // 确保data是数组
            const scripts = Array.isArray(data) ? data : [];
            const script = scripts.find(s => s.path === path);
            if (script) {
                const content = `
                    <div class="form-group">
                        <label>脚本内容:</label>
                        <textarea readonly>${script.content || '无法读取脚本内容'}</textarea>
                    </div>
                    <div class="form-actions">
                        <button onclick="closeDialog()" class="btn-secondary">关闭</button>
                    </div>
                `;
                showDialog('查看脚本', content);
            } else {
                showNotification('未找到指定的脚本', 'error');
            }
        })
        .catch(error => {
            showNotification('查看脚本失败: ' + error.message, 'error');
        });
}

function executeScript(path) {
    if (!confirm('确定要执行此脚本吗？')) {
        return;
    }
    
    fetch('/api/docker-scripts/execute', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ path: path })
    })
    .then(response => response.json())
    .then(data => {
        const statusClass = data.status === 'success' ? 'status-active' : 'status-inactive';
        const statusText = data.status === 'success' ? '执行成功' : '执行失败';
        
        const content = `
            <div class="form-group">
                <label>执行状态:</label>
                <span class="${statusClass}" style="font-weight: bold;">${statusText}</span>
            </div>
            <div class="form-group">
                <label>执行时间:</label>
                <span>${new Date(data.executed_at).toLocaleString('zh-CN')}</span>
            </div>
            <div class="form-group">
                <label>执行耗时:</label>
                <span>${data.duration}ms</span>
            </div>
            <div class="form-group">
                <label>执行结果:</label>
                <textarea readonly style="height: 300px; font-family: monospace; white-space: pre-wrap; word-break: break-all;">${data.output}</textarea>
            </div>
            <div class="form-actions">
                <button onclick="closeDialog()" class="btn-secondary">关闭</button>
            </div>
        `;
        showDialog('脚本执行结果', content);
        
        // 显示执行结果通知
        const notificationType = data.status === 'success' ? 'success' : 'error';
        const message = data.status === 'success' ? '脚本执行成功' : '脚本执行失败';
        showNotification(message, notificationType);
    })
    .catch(error => {
        showNotification('执行脚本失败: ' + error.message, 'error');
    });
}

function deleteSelectedScript() {
    if (selectedScripts.length === 0) {
        showNotification('请选择要删除的脚本', 'error');
        return;
    }
    
    if (!confirm(`确定要删除选中的 ${selectedScripts.length} 个脚本吗？`)) {
        return;
    }
    
    selectedScripts.forEach(path => {
        fetch('/api/docker-scripts', {
            method: 'DELETE',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ path: path })
        })
        .then(response => {
            if (response.ok) {
                showNotification('删除成功', 'success');
            } else {
                throw new Error('删除失败');
            }
        })
        .catch(error => {
            showNotification('删除失败: ' + error.message, 'error');
        });
    });
    
    setTimeout(() => {
        loadDockerScripts();
        selectedScripts = [];
    }, 1000);
}

function viewExecutionLogs(scriptId, scriptName) {
    if (scriptId === 0) {
        showNotification('该脚本暂无执行日志', 'info');
        return;
    }
    
    fetch(`/api/docker-scripts/logs?script_id=${scriptId}&limit=20`)
        .then(response => response.json())
        .then(data => {
            displayExecutionLogs(data, scriptName);
        })
        .catch(error => {
            showNotification('获取执行日志失败: ' + error.message, 'error');
        });
}

function displayExecutionLogs(logs, scriptName) {
    const container = document.createElement('div');
    container.style.maxHeight = '500px';
    container.style.overflowY = 'auto';
    
    if (logs.length === 0) {
        container.innerHTML = '<p style="text-align: center; color: #666; padding: 20px;">暂无执行日志</p>';
    } else {
        logs.forEach(log => {
            const logItem = document.createElement('div');
            logItem.style.borderBottom = '1px solid #eee';
            logItem.style.padding = '15px';
            logItem.style.marginBottom = '10px';
            
            const header = document.createElement('div');
            header.style.display = 'flex';
            header.style.justifyContent = 'space-between';
            header.style.alignItems = 'center';
            header.style.marginBottom = '10px';
            
            const status = document.createElement('span');
            status.textContent = log.status === 'success' ? '成功' : '失败';
            status.style.color = log.status === 'success' ? '#27ae60' : '#e74c3c';
            status.style.fontWeight = 'bold';
            
            const time = document.createElement('span');
            time.textContent = new Date(log.executed_at).toLocaleString('zh-CN');
            time.style.color = '#666';
            time.style.fontSize = '12px';
            
            const duration = document.createElement('span');
            duration.textContent = `耗时: ${log.duration_ms}ms`;
            duration.style.color = '#666';
            duration.style.fontSize = '12px';
            
            header.appendChild(status);
            header.appendChild(time);
            header.appendChild(duration);
            
            const output = document.createElement('pre');
            output.textContent = log.output || '无输出';
            output.style.backgroundColor = '#f8f9fa';
            output.style.padding = '10px';
            output.style.borderRadius = '4px';
            output.style.fontSize = '12px';
            output.style.maxHeight = '200px';
            output.style.overflowY = 'auto';
            output.style.whiteSpace = 'pre-wrap';
            output.style.wordBreak = 'break-all';
            
            logItem.appendChild(header);
            logItem.appendChild(output);
            container.appendChild(logItem);
        });
    }
    
    const content = `
        <div class="form-group">
            <label>脚本名称: ${scriptName}</label>
        </div>
        <div class="form-group">
            <label>执行日志:</label>
            <div id="execution-logs-container"></div>
        </div>
        <div class="form-actions">
            <button onclick="refreshExecutionLogs(${logs.length > 0 ? logs[0].script_id : 0}, '${scriptName}')" class="btn-primary">刷新</button>
            <button onclick="closeDialog()" class="btn-secondary">关闭</button>
        </div>
    `;
    
    showDialog('执行日志', content);
    
    // 添加日志内容到容器
    const logsContainer = document.getElementById('execution-logs-container');
    logsContainer.appendChild(container);
}

function refreshExecutionLogs(scriptId, scriptName) {
    if (scriptId === 0) return;
    
    fetch(`/api/docker-scripts/logs?script_id=${scriptId}&limit=20`)
        .then(response => response.json())
        .then(data => {
            // 清空现有内容
            const logsContainer = document.getElementById('execution-logs-container');
            logsContainer.innerHTML = '';
            
            // 重新渲染日志
            const container = document.createElement('div');
            container.style.maxHeight = '500px';
            container.style.overflowY = 'auto';
            
            if (data.length === 0) {
                container.innerHTML = '<p style="text-align: center; color: #666; padding: 20px;">暂无执行日志</p>';
            } else {
                data.forEach(log => {
                    const logItem = document.createElement('div');
                    logItem.style.borderBottom = '1px solid #eee';
                    logItem.style.padding = '15px';
                    logItem.style.marginBottom = '10px';
                    
                    const header = document.createElement('div');
                    header.style.display = 'flex';
                    header.style.justifyContent = 'space-between';
                    header.style.alignItems = 'center';
                    header.style.marginBottom = '10px';
                    
                    const status = document.createElement('span');
                    status.textContent = log.status === 'success' ? '成功' : '失败';
                    status.style.color = log.status === 'success' ? '#27ae60' : '#e74c3c';
                    status.style.fontWeight = 'bold';
                    
                    const time = document.createElement('span');
                    time.textContent = new Date(log.executed_at).toLocaleString('zh-CN');
                    time.style.color = '#666';
                    time.style.fontSize = '12px';
                    
                    const duration = document.createElement('span');
                    duration.textContent = `耗时: ${log.duration_ms}ms`;
                    duration.style.color = '#666';
                    duration.style.fontSize = '12px';
                    
                    header.appendChild(status);
                    header.appendChild(time);
                    header.appendChild(duration);
                    
                    const output = document.createElement('pre');
                    output.textContent = log.output || '无输出';
                    output.style.backgroundColor = '#f8f9fa';
                    output.style.padding = '10px';
                    output.style.borderRadius = '4px';
                    output.style.fontSize = '12px';
                    output.style.maxHeight = '200px';
                    output.style.overflowY = 'auto';
                    output.style.whiteSpace = 'pre-wrap';
                    output.style.wordBreak = 'break-all';
                    
                    logItem.appendChild(header);
                    logItem.appendChild(output);
                    container.appendChild(logItem);
                });
            }
            
            logsContainer.appendChild(container);
            showNotification('日志已刷新', 'success');
        })
        .catch(error => {
            showNotification('刷新执行日志失败: ' + error.message, 'error');
        });
}

// 端口管理功能
function loadPorts() {
    fetch('/api/ports')
        .then(response => response.json())
        .then(data => {
            displayPorts(data);
        })
        .catch(error => {
            showNotification('加载端口信息失败: ' + error.message, 'error');
        });
}

function displayPorts(ports) {
    const container = document.getElementById('port-list-content');
    container.innerHTML = '';
    
    ports.forEach(port => {
        const portItem = document.createElement('div');
        portItem.className = 'port-item';
        
        const portNum = document.createElement('span');
        portNum.textContent = port.port;
        
        const process = document.createElement('span');
        process.textContent = port.process;
        
        const pid = document.createElement('span');
        pid.textContent = port.pid;
        
        const status = document.createElement('span');
        status.textContent = port.listening ? t('listening') : t('not_listening');
        status.className = port.listening ? 'status-active' : 'status-inactive';
        
        portItem.appendChild(portNum);
        portItem.appendChild(process);
        portItem.appendChild(pid);
        portItem.appendChild(status);
        
        container.appendChild(portItem);
    });
}

function refreshPorts() {
    loadPorts();
    showNotification('端口信息已刷新', 'success');
}

// 服务管理功能
function loadServices() {
    fetch('/api/services')
        .then(response => response.json())
        .then(data => {
            displayServices(data);
        })
        .catch(error => {
            showNotification('加载服务信息失败: ' + error.message, 'error');
        });
}

function displayServices(services) {
    const container = document.getElementById('service-list-content');
    container.innerHTML = '';
    
    services.forEach(service => {
        const serviceItem = document.createElement('div');
        serviceItem.className = 'service-item';
        
        const name = document.createElement('span');
        name.textContent = service.name;
        
        const status = document.createElement('span');
        status.textContent = service.status;
        status.className = service.status === 'active' || service.status === 'running' ? 'status-active' : 'status-inactive';
        
        const pid = document.createElement('span');
        pid.textContent = service.pid;
        
        const memory = document.createElement('span');
        memory.textContent = service.memory;
        
        const cpu = document.createElement('span');
        cpu.textContent = service.cpu;
        
        const actions = document.createElement('span');
        actions.className = 'service-actions';
        actions.innerHTML = `
            <button onclick="startService('${service.name}')" class="btn-start">启动</button>
            <button onclick="stopService('${service.name}')" class="btn-stop">停止</button>
            <button onclick="restartService('${service.name}')" class="btn-restart">重启</button>
        `;
        
        serviceItem.appendChild(name);
        serviceItem.appendChild(status);
        serviceItem.appendChild(pid);
        serviceItem.appendChild(memory);
        serviceItem.appendChild(cpu);
        serviceItem.appendChild(actions);
        
        container.appendChild(serviceItem);
    });
}

function refreshServices() {
    loadServices();
    showNotification('服务信息已刷新', 'success');
}

function startService(serviceName) {
    fetch('/api/services/start', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ name: serviceName })
    })
    .then(response => {
        if (response.ok) {
            showNotification('服务启动成功', 'success');
            loadServices();
        } else {
            throw new Error('启动失败');
        }
    })
    .catch(error => {
        showNotification('启动服务失败: ' + error.message, 'error');
    });
}

function stopService(serviceName) {
    fetch('/api/services/stop', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ name: serviceName })
    })
    .then(response => {
        if (response.ok) {
            showNotification('服务停止成功', 'success');
            loadServices();
        } else {
            throw new Error('停止失败');
        }
    })
    .catch(error => {
        showNotification('停止服务失败: ' + error.message, 'error');
    });
}

function restartService(serviceName) {
    fetch('/api/services/restart', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ name: serviceName })
    })
    .then(response => {
        if (response.ok) {
            showNotification('服务重启成功', 'success');
            loadServices();
        } else {
            throw new Error('重启失败');
        }
    })
    .catch(error => {
        showNotification('重启服务失败: ' + error.message, 'error');
    });
}

// 对话框功能
function showDialog(title, content) {
    document.getElementById('dialog-title').textContent = title;
    document.getElementById('dialog-content').innerHTML = content;
    document.getElementById('dialog').style.display = 'flex';
}

function closeDialog() {
    document.getElementById('dialog').style.display = 'none';
}

// 通知功能
function showNotification(message, type = 'info') {
    const notification = document.getElementById('notification');
    notification.textContent = message;
    notification.className = `notification ${type}`;
    notification.style.display = 'block';
    
    setTimeout(() => {
        notification.style.display = 'none';
    }, 3000);
}

// 容器管理功能
function loadContainers() {
    fetch('/api/containers')
        .then(response => response.json())
        .then(data => {
            displayContainers(data);
        })
        .catch(error => {
            showNotification('加载容器列表失败: ' + error.message, 'error');
        });
}

function displayContainers(containers) {
    const container = document.getElementById('container-list-content');
    container.innerHTML = '';
    
    containers.forEach(containerInfo => {
        const containerItem = document.createElement('div');
        containerItem.className = 'container-item';
        
        const id = document.createElement('span');
        id.textContent = containerInfo.id.substring(0, 12);
        id.title = containerInfo.id;
        
        const name = document.createElement('span');
        name.textContent = containerInfo.name;
        
        const image = document.createElement('span');
        image.textContent = containerInfo.image;
        
        const status = document.createElement('span');
        status.textContent = containerInfo.status;
        status.className = containerInfo.status.includes('Up') ? 'status-active' : 'status-inactive';
        
        const ports = document.createElement('span');
        ports.textContent = containerInfo.ports || '-';
        
        const created = document.createElement('span');
        created.textContent = containerInfo.created;
        
        const actions = document.createElement('span');
        actions.className = 'container-actions';
        actions.innerHTML = `
            <button onclick="startContainer('${containerInfo.id}')" class="btn-start">启动</button>
            <button onclick="stopContainer('${containerInfo.id}')" class="btn-stop">停止</button>
            <button onclick="restartContainer('${containerInfo.id}')" class="btn-restart">重启</button>
            <button onclick="viewContainerLogs('${containerInfo.id}')" class="btn-execute">日志</button>
            <button onclick="removeContainer('${containerInfo.id}')" class="btn-stop">删除</button>
        `;
        
        containerItem.appendChild(id);
        containerItem.appendChild(name);
        containerItem.appendChild(image);
        containerItem.appendChild(status);
        containerItem.appendChild(ports);
        containerItem.appendChild(created);
        containerItem.appendChild(actions);
        
        container.appendChild(containerItem);
    });
}

function refreshContainers() {
    loadContainers();
    showNotification('容器列表已刷新', 'success');
}

function startContainer(containerId) {
    fetch('/api/containers/start', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ id: containerId })
    })
    .then(response => {
        if (response.ok) {
            showNotification('容器启动成功', 'success');
            loadContainers();
        } else {
            throw new Error('启动失败');
        }
    })
    .catch(error => {
        showNotification('启动容器失败: ' + error.message, 'error');
    });
}

function stopContainer(containerId) {
    if (!confirm('确定要停止此容器吗？')) {
        return;
    }
    
    fetch('/api/containers/stop', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ id: containerId })
    })
    .then(response => {
        if (response.ok) {
            showNotification('容器停止成功', 'success');
            loadContainers();
        } else {
            throw new Error('停止失败');
        }
    })
    .catch(error => {
        showNotification('停止容器失败: ' + error.message, 'error');
    });
}

function restartContainer(containerId) {
    fetch('/api/containers/restart', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ id: containerId })
    })
    .then(response => {
        if (response.ok) {
            showNotification('容器重启成功', 'success');
            loadContainers();
        } else {
            throw new Error('重启失败');
        }
    })
    .catch(error => {
        showNotification('重启容器失败: ' + error.message, 'error');
    });
}

function removeContainer(containerId) {
    if (!confirm('确定要删除此容器吗？此操作不可恢复！')) {
        return;
    }
    
    fetch('/api/containers/remove', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ id: containerId })
    })
    .then(response => {
        if (response.ok) {
            showNotification('容器删除成功', 'success');
            loadContainers();
        } else {
            throw new Error('删除失败');
        }
    })
    .catch(error => {
        showNotification('删除容器失败: ' + error.message, 'error');
    });
}

function viewContainerLogs(containerId) {
    fetch('/api/containers/logs', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ id: containerId })
    })
    .then(response => response.json())
    .then(data => {
        const content = `
            <div class="form-group">
                <label>容器日志:</label>
                <textarea readonly style="height: 400px; font-family: monospace;">${data.logs}</textarea>
            </div>
            <div class="form-actions">
                <button onclick="closeDialog()" class="btn-secondary">关闭</button>
            </div>
        `;
        showDialog('容器日志', content);
    })
    .catch(error => {
        showNotification('获取容器日志失败: ' + error.message, 'error');
    });
}

// 镜像管理功能
function loadImages() {
    fetch('/api/images')
        .then(response => response.json())
        .then(data => {
            displayImages(data);
        })
        .catch(error => {
            showNotification('加载镜像列表失败: ' + error.message, 'error');
        });
}

function displayImages(images) {
    const container = document.getElementById('image-list-content');
    container.innerHTML = '';
    
    images.forEach(imageInfo => {
        const imageItem = document.createElement('div');
        imageItem.className = 'image-item';
        
        const id = document.createElement('span');
        id.textContent = imageInfo.id.substring(0, 12);
        id.title = imageInfo.id;
        
        const repository = document.createElement('span');
        repository.textContent = imageInfo.repository;
        
        const tag = document.createElement('span');
        tag.textContent = imageInfo.tag;
        
        const size = document.createElement('span');
        size.textContent = imageInfo.size;
        
        const created = document.createElement('span');
        created.textContent = imageInfo.created;
        
        const actions = document.createElement('span');
        actions.className = 'image-actions';
        actions.innerHTML = `
            <button onclick="removeImage('${imageInfo.id}')" class="btn-stop">删除</button>
        `;
        
        imageItem.appendChild(id);
        imageItem.appendChild(repository);
        imageItem.appendChild(tag);
        imageItem.appendChild(size);
        imageItem.appendChild(created);
        imageItem.appendChild(actions);
        
        container.appendChild(imageItem);
    });
}

function refreshImages() {
    loadImages();
    showNotification('镜像列表已刷新', 'success');
}

function showPullImageDialog() {
    const content = `
        <div class="form-group">
            <label for="image-name">镜像名称:</label>
            <input type="text" id="image-name" placeholder="例如: nginx:latest, ubuntu:20.04">
        </div>
        <div class="form-actions">
            <button onclick="pullImage()" class="btn-primary">拉取</button>
            <button onclick="closeDialog()" class="btn-secondary">取消</button>
        </div>
    `;
    showDialog('拉取镜像', content);
}

function pullImage() {
    const imageName = document.getElementById('image-name').value.trim();
    if (!imageName) {
        showNotification('请输入镜像名称', 'error');
        return;
    }
    
    fetch('/api/images/pull', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ image: imageName })
    })
    .then(response => response.json())
    .then(data => {
        const content = `
            <div class="form-group">
                <label>拉取结果:</label>
                <textarea readonly style="height: 400px; font-family: monospace;">${data.output}</textarea>
            </div>
            <div class="form-actions">
                <button onclick="closeDialog(); loadImages();" class="btn-secondary">关闭</button>
            </div>
        `;
        showDialog('镜像拉取结果', content);
    })
    .catch(error => {
        showNotification('拉取镜像失败: ' + error.message, 'error');
    });
}

function removeImage(imageId) {
    if (!confirm('确定要删除此镜像吗？此操作不可恢复！')) {
        return;
    }
    
    fetch('/api/images/remove', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ id: imageId })
    })
    .then(response => {
        if (response.ok) {
            showNotification('镜像删除成功', 'success');
            loadImages();
        } else {
            throw new Error('删除失败');
        }
    })
    .catch(error => {
        showNotification('删除镜像失败: ' + error.message, 'error');
    });
}

function pruneImages() {
    if (!confirm('确定要清理所有未使用的镜像吗？此操作不可恢复！')) {
        return;
    }
    
    fetch('/api/images/prune', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        }
    })
    .then(response => response.json())
    .then(data => {
        const content = `
            <div class="form-group">
                <label>清理结果:</label>
                <textarea readonly style="height: 300px; font-family: monospace;">${data.output}</textarea>
            </div>
            <div class="form-actions">
                <button onclick="closeDialog(); loadImages();" class="btn-secondary">关闭</button>
            </div>
        `;
        showDialog('镜像清理结果', content);
    })
    .catch(error => {
        showNotification('清理镜像失败: ' + error.message, 'error');
    });
}

// 点击对话框外部关闭
document.getElementById('dialog').addEventListener('click', function(e) {
    if (e.target === this) {
        closeDialog();
    }
});

// 系统监控相关变量
let monitoringWebSocket = null;
let isMonitoring = false;

// 图表相关变量
let cpuChart = null;
let memoryChart = null;
let chartUpdateInterval = null;

// 前端历史数据存储
let cpuHistoryData = [];
let memoryHistoryData = [];
let maxDataPoints = 900; // 30分钟的数据 (30*60/2=900个点)

// 格式化文件大小
function formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// 切换监控状态
function toggleMonitoring() {
    const btn = document.getElementById('monitoring-toggle-btn');
    const status = document.getElementById('monitoring-status');
    
    if (isMonitoring) {
        stopMonitoring();
        btn.textContent = t('start_monitoring');
        btn.classList.remove('active');
        status.textContent = t('disconnected');
        status.className = 'status-disconnected';
    } else {
        startMonitoring();
        btn.textContent = t('stop_monitoring');
        btn.classList.add('active');
        status.textContent = t('connecting');
        status.className = 'status-connected';
    }
}

// 开始监控
function startMonitoring() {
    if (monitoringWebSocket) {
        monitoringWebSocket.close();
    }
    
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    monitoringWebSocket = new WebSocket(wsUrl);
    
    monitoringWebSocket.onopen = function() {
        isMonitoring = true;
        const status = document.getElementById('monitoring-status');
        status.textContent = t('connected');
        status.className = 'status-connected';
        console.log('WebSocket 连接已建立');
        
        // 清空历史数据并开始图表更新
        cpuHistoryData = [];
        memoryHistoryData = [];
        startChartUpdates();
    };
    
    monitoringWebSocket.onmessage = function(event) {
        const data = JSON.parse(event.data);
        updateMonitoringDisplay(data);
    };
    
    monitoringWebSocket.onclose = function() {
        isMonitoring = false;
        const status = document.getElementById('monitoring-status');
        status.textContent = t('disconnected');
        status.className = 'status-disconnected';
        console.log('WebSocket 连接已断开');
        
        // 停止图表更新
        stopChartUpdates();
    };
    
    monitoringWebSocket.onerror = function(error) {
        console.error('WebSocket 错误:', error);
        const status = document.getElementById('monitoring-status');
        status.textContent = t('connection_error');
        status.className = 'status-disconnected';
        
        // 停止图表更新
        stopChartUpdates();
    };
}

// 停止监控
function stopMonitoring() {
    if (monitoringWebSocket) {
        monitoringWebSocket.close();
        monitoringWebSocket = null;
    }
    isMonitoring = false;
    
    // 停止图表更新
    stopChartUpdates();
}

// 更新监控显示
function updateMonitoringDisplay(data) {
    // 更新CPU使用率
    const cpuUsage = data.cpu_usage.toFixed(1);
    document.getElementById('cpu-usage').textContent = cpuUsage + '%';
    const cpuProgress = document.getElementById('cpu-progress');
    cpuProgress.style.width = cpuUsage + '%';
    
    // 根据使用率设置进度条颜色
    if (cpuUsage > 80) {
        cpuProgress.className = 'progress-fill danger';
    } else if (cpuUsage > 60) {
        cpuProgress.className = 'progress-fill warning';
    } else {
        cpuProgress.className = 'progress-fill';
    }
    
    // 更新内存使用
    const memoryUsed = formatFileSize(data.memory_usage.used);
    const memoryTotal = formatFileSize(data.memory_usage.total);
    const memoryPercent = data.memory_usage.percent.toFixed(1);
    document.getElementById('memory-usage').textContent = `${memoryUsed} / ${memoryTotal}`;
    document.getElementById('memory-percent').textContent = memoryPercent + '%';
    const memoryProgress = document.getElementById('memory-progress');
    memoryProgress.style.width = memoryPercent + '%';
    
    if (memoryPercent > 80) {
        memoryProgress.className = 'progress-fill danger';
    } else if (memoryPercent > 60) {
        memoryProgress.className = 'progress-fill warning';
    } else {
        memoryProgress.className = 'progress-fill';
    }
    
    // 更新磁盘使用
    const diskUsed = formatFileSize(data.disk_usage.used);
    const diskTotal = formatFileSize(data.disk_usage.total);
    const diskPercent = data.disk_usage.percent.toFixed(1);
    document.getElementById('disk-usage').textContent = `${diskUsed} / ${diskTotal}`;
    document.getElementById('disk-percent').textContent = diskPercent + '%';
    const diskProgress = document.getElementById('disk-progress');
    diskProgress.style.width = diskPercent + '%';
    
    if (diskPercent > 80) {
        diskProgress.className = 'progress-fill danger';
    } else if (diskPercent > 60) {
        diskProgress.className = 'progress-fill warning';
    } else {
        diskProgress.className = 'progress-fill';
    }
    
    // 更新系统运行时间
    document.getElementById('system-uptime').textContent = data.uptime;
    
    // 更新最后更新时间
    const updateTime = new Date(data.timestamp).toLocaleString();
    document.getElementById('last-update').textContent = updateTime;
    
    // 存储历史数据并更新图表
    storeHistoricalData(data);
    updateChartsFromLocalData();
}

// 存储历史数据
function storeHistoricalData(data) {
    const now = new Date();
    
    // 添加CPU数据 - 确保时间戳格式兼容G2
    cpuHistoryData.push({
        timestamp: now,
        value: parseFloat(data.cpu_usage.toFixed(1))
    });
    
    // 添加内存数据 - 确保时间戳格式兼容G2
    memoryHistoryData.push({
        timestamp: now,
        value: parseFloat(data.memory_usage.percent.toFixed(1))
    });
    
    // 保持数据点数量在限制范围内
    if (cpuHistoryData.length > maxDataPoints) {
        cpuHistoryData = cpuHistoryData.slice(-maxDataPoints);
    }
    if (memoryHistoryData.length > maxDataPoints) {
        memoryHistoryData = memoryHistoryData.slice(-maxDataPoints);
    }
}

// 从本地数据更新图表
function updateChartsFromLocalData() {
    const timeRange = document.getElementById('time-range').value;
    const filteredCpuData = filterDataByTimeRange(cpuHistoryData, timeRange);
    const filteredMemoryData = filterDataByTimeRange(memoryHistoryData, timeRange);
    
    if (cpuChart && filteredCpuData.length > 0) {
        cpuChart.changeData(filteredCpuData);
    }
    if (memoryChart && filteredMemoryData.length > 0) {
        memoryChart.changeData(filteredMemoryData);
    }
}

// 根据时间范围过滤数据
function filterDataByTimeRange(data, timeRange) {
    if (!data || data.length === 0) return [];
    
    const now = new Date();
    let cutoffTime;
    
    switch (timeRange) {
        case '1min':
            cutoffTime = new Date(now.getTime() - 1 * 60 * 1000);
            break;
        case '5min':
            cutoffTime = new Date(now.getTime() - 5 * 60 * 1000);
            break;
        case '30min':
            cutoffTime = new Date(now.getTime() - 30 * 60 * 1000);
            break;
        default:
            cutoffTime = new Date(now.getTime() - 5 * 60 * 1000);
    }
    
    // 确保时间戳格式正确
    return data.filter(item => {
        const itemTime = new Date(item.timestamp);
        return itemTime >= cutoffTime && !isNaN(itemTime.getTime());
    });
}

// 页面卸载时关闭WebSocket连接
window.addEventListener('beforeunload', function() {
    stopMonitoring();
    stopChartUpdates();
});

// 图表相关函数
function initCharts() {
  /* ---------- CPU 图表 ---------- */
  cpuChart = new G2.Chart({
    container: 'cpu-chart',
    autoFit: true,
    height: 300,
  });

  cpuChart.data([]);

  cpuChart
    .scale('timestamp', { type: 'time', tickCount: 8 })
    .axis('timestamp', { title: '时间', labelFormatter: (d) => d.toTimeString().slice(0, 8) })
    .scale('value', { domainMin: 0, domainMax: 100, nice: true })
    .axis('value', { title: 'CPU使用率 (%)' });

  cpuChart
    .line()
    .encode('x', 'timestamp')
    .encode('y', 'value')
    .animate(false);

  cpuChart
    .point()
    .encode('x', 'timestamp')
    .encode('y', 'value')
    .encode('shape', 'circle')
    .encode('size', 3);

  cpuChart.render();

  /* ---------- 内存 图表 ---------- */
  memoryChart = new G2.Chart({
    container: 'memory-chart',
    autoFit: true,
    height: 300,
  });

  memoryChart.data([]);

  memoryChart
    .scale('timestamp', { type: 'time', tickCount: 8 })
    .axis('timestamp', { title: '时间', labelFormatter: (d) => d.toTimeString().slice(0, 8) })
    .scale('value', { domainMin: 0, domainMax: 100, nice: true })
    .axis('value', { title: '内存使用率 (%)' });

  memoryChart
    .line()
    .encode('x', 'timestamp')
    .encode('y', 'value')
    .animate(false);

  memoryChart
    .point()
    .encode('x', 'timestamp')
    .encode('y', 'value')
    .encode('shape', 'circle')
    .encode('size', 3);

  memoryChart.render();
}

function updateCharts() {
    updateChartsFromLocalData();
}

function refreshCharts() {
    updateCharts();
    showNotification('图表已刷新', 'success');
}

function startChartUpdates() {
    // 每5秒更新一次图表
    chartUpdateInterval = setInterval(() => {
        updateChartsFromLocalData();
    }, 5000);
}

function stopChartUpdates() {
    if (chartUpdateInterval) {
        clearInterval(chartUpdateInterval);
        chartUpdateInterval = null;
    }
}

// 在页面加载时初始化图表（在原有的DOMContentLoaded中已添加）