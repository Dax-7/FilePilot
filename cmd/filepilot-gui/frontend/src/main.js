import './style.css';

const api = () => window.go?.desktop?.App;
const runtime = () => window.runtime;

const state = {
  mode: 'home',
  sendPath: '',
  receiveFolder: '',
  lastReceivePath: ''
};

const el = (id) => document.getElementById(id);

function showMode(mode) {
  state.mode = mode;
  el('modeGrid').classList.toggle('hidden', mode !== 'home');
  el('sendPanel').classList.toggle('hidden', mode !== 'send');
  el('receivePanel').classList.toggle('hidden', mode !== 'receive');
}

function setGlobalStatus(text) {
  el('globalStatus').textContent = text;
}

function appendLog(target, text, kind = '') {
  const item = document.createElement('li');
  item.textContent = text;
  if (kind) item.classList.add(kind);
  target.prepend(item);
}

function setSendPath(path) {
  state.sendPath = path || '';
  el('sendPathLabel').textContent = state.sendPath || 'No file or folder selected';
  el('sendPathHint').textContent = state.sendPath ? 'Ready to send.' : 'Use one of the selectors below.';
}

function handleEvent(event) {
  const target = state.mode === 'receive' ? el('receiveLog') : el('sendLog');
  if (event.session_id) {
    el('sessionRow').classList.remove('hidden');
    el('sessionCode').textContent = event.session_id;
  }
  const label = event.type || 'status';
  const message = event.message || label;
  setGlobalStatus(label);
  appendLog(target, message, event.type === 'error' ? 'error' : '');
}

window.addEventListener('DOMContentLoaded', () => {
  showMode('home');
  api().DefaultReceiveFolder().then((result) => {
    if (!result?.ok || !result.path) return;
    state.receiveFolder = result.path;
    el('receiveFolder').value = result.path;
  });
  runtime()?.EventsOn?.('transfer:event', handleEvent);
  runtime()?.OnFileDrop?.((_x, _y, paths) => {
    if (state.mode !== 'send' || !paths || paths.length === 0) return;
    setSendPath(paths[0]);
    if (paths.length > 1) {
      appendLog(el('sendLog'), 'FilePilot sends one file or folder per transfer. The first dropped path was selected.');
    }
  }, true);

  el('sendMode').addEventListener('click', () => showMode('send'));
  el('receiveMode').addEventListener('click', () => showMode('receive'));
  el('sendBack').addEventListener('click', () => showMode('home'));
  el('receiveBack').addEventListener('click', () => showMode('home'));

  el('chooseFile').addEventListener('click', async () => {
    const path = await api().ChooseSendFile();
    setSendPath(path);
  });

  el('chooseFolder').addEventListener('click', async () => {
    const path = await api().ChooseSendFolder();
    setSendPath(path);
  });

  el('sendDrop').addEventListener('dragover', (event) => {
    event.preventDefault();
    el('sendDrop').classList.add('dragging');
  });
  el('sendDrop').addEventListener('dragleave', () => el('sendDrop').classList.remove('dragging'));
  el('sendDrop').addEventListener('drop', (event) => {
    event.preventDefault();
    el('sendDrop').classList.remove('dragging');
  });

  el('startSend').addEventListener('click', async () => {
    el('sendLog').replaceChildren();
    setGlobalStatus('started');
    const result = await api().StartSend(state.sendPath);
    if (!result.ok) {
      setGlobalStatus('failed');
      appendLog(el('sendLog'), result.error || 'Send failed.', 'error');
      return;
    }
    setGlobalStatus('completed');
    el('sessionRow').classList.remove('hidden');
    el('sessionCode').textContent = result.session_id;
    appendLog(el('sendLog'), 'Send completed.');
  });

  el('copyCode').addEventListener('click', async () => {
    const code = el('sessionCode').textContent;
    if (code) await navigator.clipboard.writeText(code);
  });

  el('chooseReceiveFolder').addEventListener('click', async () => {
    const path = await api().ChooseReceiveFolder();
    state.receiveFolder = path || '';
    el('receiveFolder').value = state.receiveFolder;
  });

  el('startReceive').addEventListener('click', async () => {
    el('receiveLog').replaceChildren();
    setGlobalStatus('started');
    const result = await api().StartReceive(el('receiveCode').value.trim(), state.receiveFolder);
    if (!result.ok) {
      setGlobalStatus('failed');
      appendLog(el('receiveLog'), result.error || 'Receive failed.', 'error');
      return;
    }
    setGlobalStatus('completed');
    state.lastReceivePath = result.path || state.receiveFolder;
    el('openFolder').classList.remove('hidden');
    appendLog(el('receiveLog'), 'Receive completed.');
  });

  el('openFolder').addEventListener('click', async () => {
    await api().OpenFolder(state.receiveFolder || state.lastReceivePath);
  });
});
