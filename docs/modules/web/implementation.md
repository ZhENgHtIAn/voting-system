# Web 模块实现文档（Phase 3）

## 文件清单
- `web/index.html`
- `web/app.js`

## 实现说明
- `index.html` 提供投票页面骨架，渲染话题列表容器与错误提示区域。
- `app.js` 负责：
  - 页面初始化时调用 `GET /api/results`
  - 点击投票按钮时调用 `POST /api/vote`
  - 将返回结果实时渲染到页面

## 函数定位
- `fetchResults()`  
  - 文件：`web/app.js`  
  - 职责：请求当前投票结果。
- `submitVote(topicName)`  
  - 文件：`web/app.js`  
  - 职责：提交单次投票请求。
- `renderResults(results)`  
  - 文件：`web/app.js`  
  - 职责：将结果渲染为按钮列表与票数。
- `showError(message)`  
  - 文件：`web/app.js`  
  - 职责：展示错误信息。
- `clearError()`  
  - 文件：`web/app.js`  
  - 职责：清空错误信息。
- `bootstrap()`  
  - 文件：`web/app.js`  
  - 职责：页面启动入口，初始化加载结果。
