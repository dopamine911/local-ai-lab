# ComfyUI for zaphod's R9700s (RDNA4, gfx1201), built from official pieces:
#   ComfyUI release tag + PyTorch's ROCm wheels (they bundle the ROCm runtime; nothing ROCm on the host).
# Image tag: comfyui-zaphod:<COMFYUI_VERSION>-torch<TORCH>-rocm<ROCM>
FROM python:3.13-slim-trixie

ARG COMFYUI_VERSION=v0.43.0
ARG TORCH=2.15.0
ARG ROCM=7.14

RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates libgl1 libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

# Non-root user; uid 1000 so files in the mounted folders belong to daniel
RUN useradd -m -u 1000 comfy
USER comfy
WORKDIR /home/comfy

RUN git clone --depth 1 --branch ${COMFYUI_VERSION} https://github.com/Comfy-Org/ComfyUI.git \
    && python -m venv venv \
    && venv/bin/pip install --no-cache-dir --index-url https://download.pytorch.org/whl/rocm${ROCM} \
         "torch==${TORCH}+rocm${ROCM}" torchvision \
    && venv/bin/pip install --no-cache-dir -r ComfyUI/requirements.txt \
    && venv/bin/python -c "import torch; print('torch', torch.__version__, 'hip', torch.version.hip)"

WORKDIR /home/comfy/ComfyUI
EXPOSE 8188
ENTRYPOINT ["/home/comfy/venv/bin/python", "main.py", "--listen", "127.0.0.1", "--port", "8188"]
