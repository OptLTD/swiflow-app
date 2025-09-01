<script setup lang="ts">
import { ref, computed } from 'vue';
import { onMounted, onUnmounted } from 'vue';
import { useAppStore } from '@/stores/app';

const props = defineProps({
  duration: {
    type: Number,
    default: 8 // 默认8秒
  }
});

const app = useAppStore();

const emit = defineEmits(['close']);

// 倒计时相关
const remainingTime = ref(props.duration);
const timerInterval = ref(null as NodeJS.Timeout | null);

// 开始倒计时
const startCountdown = () => {
  remainingTime.value = props.duration;
  timerInterval.value = setInterval(() => {
    if (remainingTime.value > 0) {
      remainingTime.value--;
    } else {
      clearInterval(timerInterval.value as NodeJS.Timeout);
      timerInterval.value = null;
      emit('close');
    }
  }, 1000);
};

// 手动关闭
const handleClose = () => {
  if (timerInterval.value) {
    clearInterval(timerInterval.value);
    timerInterval.value = null;
  }
  emit('close');
};

const showText = computed(() => {
  return !!app.display.epigraphText;
});

// 新增：多行文本分割
const fireworksLines = computed(() => {
  if (!app.display.epigraphText) return [];
  // 支持\n和<br/>分割
  return app.display.epigraphText
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/\//ig, '\n').split(/\n/)
    .filter(line => line.trim().length > 0);
});

const canvas = ref<HTMLCanvasElement | null>(null);
let animationFrameId: number | null = null;
let ctx: CanvasRenderingContext2D | null = null;
let fireworks: Firework[] = [];
let particles: Particle[] = [];
let hue = 120;
let limiterTotal = 3; // 减少限制器总数，使鼠标点击更快创建烟花
let limiterTick = 0;
let timerTotal = 40; // 减少计时器总数，使自动烟花发射更频繁
let timerTick = 0;
let mousedown = false;
let mx: number, my: number;
let canvasWidth: number, canvasHeight: number;

class Firework {
  x: number;
  y: number;
  sx: number;
  sy: number;
  tx: number;
  ty: number;
  distanceToTarget: number;
  distanceTraveled: number;
  coordinates: number[][];
  coordinateCount: number;
  angle: number;
  speed: number;
  acceleration: number;
  brightness: number;
  targetRadius: number;
  hue: number;

  constructor(sx: number, sy: number, tx: number, ty: number) {
    // 起始坐标
    this.x = sx;
    this.y = sy;
    // 起始点
    this.sx = sx;
    this.sy = sy;
    // 目标点
    this.tx = tx;
    this.ty = ty;
    // 计算距离
    this.distanceToTarget = Math.sqrt(Math.pow(this.tx - this.sx, 2) + Math.pow(this.ty - this.sy, 2));
    this.distanceTraveled = 0;
    // 跟踪过去的坐标以创建轨迹效果
    this.coordinates = [];
    this.coordinateCount = 3;
    // 填充初始坐标集合
    while (this.coordinateCount--) {
      this.coordinates.push([this.x, this.y]);
    }
    this.angle = Math.atan2(ty - sy, tx - sx);
    this.speed = 2;
    this.acceleration = 1.05;
    this.brightness = Math.random() * 50 + 50;
    this.targetRadius = 1;
    // 设置色调
    this.hue = hue;
  }

  update(index: number) {
    // 移除最旧的坐标
    this.coordinates.pop();
    // 添加当前坐标到坐标开头
    this.coordinates.unshift([this.x, this.y]);

    // 目标圆圈效果
    if (this.targetRadius < 8) {
      this.targetRadius += 0.3;
    } else {
      this.targetRadius = 1;
    }

    // 加速移动
    this.speed *= this.acceleration;

    // 计算当前速度的x和y分量
    const vx = Math.cos(this.angle) * this.speed;
    const vy = Math.sin(this.angle) * this.speed;
    // 计算已经走过的距离
    this.distanceTraveled = Math.sqrt(Math.pow(this.x - this.sx, 2) + Math.pow(this.y - this.sy, 2));

    // 如果已经到达目标，则爆炸
    if (this.distanceTraveled >= this.distanceToTarget) {
      createParticles(this.tx, this.ty, this.hue);
      // 移除烟花
      fireworks.splice(index, 1);
    } else {
      // 继续移动
      this.x += vx;
      this.y += vy;
    }
  }

  draw() {
    if (!ctx) return;
    ctx.beginPath();
    // 移动到最后记录的坐标
    ctx.moveTo(this.coordinates[this.coordinates.length - 1][0], this.coordinates[this.coordinates.length - 1][1]);
    // 画线到当前坐标
    ctx.lineTo(this.x, this.y);
    ctx.strokeStyle = `hsl(${this.hue}, 100%, ${this.brightness}%)`;
    ctx.stroke();

    // 绘制目标圆圈
    ctx.beginPath();
    ctx.arc(this.tx, this.ty, this.targetRadius, 0, Math.PI * 2);
    ctx.stroke();
  }
}

class Particle {
  x: number;
  y: number;
  coordinates: number[][];
  coordinateCount: number;
  angle: number;
  speed: number;
  friction: number;
  gravity: number;
  hue: number;
  brightness: number;
  alpha: number;
  decay: number;

  constructor(x: number, y: number, hue: number) {
    this.x = x;
    this.y = y;
    // 跟踪过去的坐标以创建轨迹效果
    this.coordinates = [];
    this.coordinateCount = 5;
    while (this.coordinateCount--) {
      this.coordinates.push([this.x, this.y]);
    }
    // 设置随机角度
    this.angle = Math.random() * Math.PI * 2;
    this.speed = Math.random() * 10 + 2;
    // 摩擦会减慢粒子
    this.friction = 0.95;
    // 重力会拉下粒子
    this.gravity = 1;
    // 设置色调
    this.hue = Math.random() * 50 + hue;
    this.brightness = Math.random() * 50 + 50;
    this.alpha = 1;
    // 设置粒子消失的速度
    this.decay = Math.random() * 0.03 + 0.015;
  }

  update(index: number) {
    // 移除最后的坐标
    this.coordinates.pop();
    // 添加当前坐标到开头
    this.coordinates.unshift([this.x, this.y]);
    // 减慢粒子
    this.speed *= this.friction;
    // 应用速度
    this.x += Math.cos(this.angle) * this.speed;
    this.y += Math.sin(this.angle) * this.speed + this.gravity;
    // 淡出粒子
    this.alpha -= this.decay;

    // 当粒子变得透明时移除它
    if (this.alpha <= this.decay) {
      particles.splice(index, 1);
    }
  }

  draw() {
    if (!ctx) return;
    ctx.beginPath();
    // 移动到最后记录的坐标
    ctx.moveTo(this.coordinates[this.coordinates.length - 1][0], this.coordinates[this.coordinates.length - 1][1]);
    // 画线到当前坐标
    ctx.lineTo(this.x, this.y);
    ctx.strokeStyle = `hsla(${this.hue}, 100%, ${this.brightness}%, ${this.alpha})`;
    ctx.stroke();
  }
}

// 创建粒子
function createParticles(x: number, y: number, hue: number) {
  // 创建50-70个粒子，增加爆炸效果
  const particleCount = Math.floor(Math.random() * 20) + 50;
  console.log('Creating particles:', particleCount);
  
  for (let i = 0; i < particleCount; i++) {
    particles.push(new Particle(x, y, hue));
  }
}

// 随机创建烟花
function createRandomFirework() {
  if (!canvas.value) return;
  
  // 创建1-3个烟花，增加密度
  const count = Math.floor(Math.random() * 3) + 1;
  
  for (let i = 0; i < count; i++) {
    // 随机起始位置（底部）
    const sx = Math.random() * canvasWidth * 0.8 + canvasWidth * 0.1;
    const sy = canvasHeight;
    
    // 随机目标位置（上半部分）
    const tx = Math.random() * canvasWidth;
    const ty = Math.random() * canvasHeight * 0.6;
    
    // 创建烟花
    fireworks.push(new Firework(sx, sy, tx, ty));
  }
  
  console.log('Created random fireworks:', count);
}

// 主循环
function loop() {
  // 检查必要条件
  if (!ctx || !canvas.value) {
    console.error('Canvas or context not available');
    return;
  }
  
  // 使用v-if后，组件只有在show为true时才会挂载，不需要再检查show属性
  console.log('Fireworks animation loop running');

  // 清除画布并绘制黑色背景
  ctx.globalCompositeOperation = 'source-over';
  ctx.fillStyle = 'rgba(0, 0, 0, 1)';
  ctx.fillRect(0, 0, canvasWidth, canvasHeight);
  
  // 设置透明度以创建拖尾效果
  ctx.globalCompositeOperation = 'lighter';

  // 循环遍历每个烟花
  for (let i = 0; i < fireworks.length; i++) {
    fireworks[i].draw();
    fireworks[i].update(i);
  }

  // 循环遍历每个粒子
  for (let i = 0; i < particles.length; i++) {
    particles[i].draw();
    particles[i].update(i);
  }

  // 随机发射烟花
  if (timerTick >= timerTotal) {
    if (!mousedown) {
      // 使用v-if后，组件只有在show为true时才会挂载，不需要再检查show属性
      createRandomFirework();
    }
    timerTick = 0;
  } else {
    timerTick++;
  }

  // 限制烟花发射频率
  if (limiterTick >= limiterTotal) {
    if (mousedown) {
      // 在鼠标位置创建烟花
      fireworks.push(new Firework(canvasWidth / 2, canvasHeight, mx, my));
      limiterTick = 0;
    }
  } else {
    limiterTick++;
  }

  // 增加色调
  hue += 0.5;

  // 请求下一帧
  // 使用v-if后，组件只有在show为true时才会挂载，不需要再检查show属性
  animationFrameId = requestAnimationFrame(loop);
}

// 调整画布大小
function resizeCanvas() {
  if (!canvas.value) return;
  canvasWidth = window.innerWidth;
  canvasHeight = window.innerHeight;
  canvas.value.width = canvasWidth;
  canvas.value.height = canvasHeight;
}

// 使用v-if后，不需要监听show属性变化
// 组件只有在show为true时才会挂载，在show为false时会被销毁
// 所有的初始化逻辑都在onMounted中处理，所有的清理逻辑都在onUnmounted中处理

onMounted(() => {
  // 启动倒计时
  startCountdown();
  
  // 确保在下一个 tick 后获取 canvas 引用
  setTimeout(() => {
    if (canvas.value) {
      console.log('Canvas element found');
      ctx = canvas.value.getContext('2d');
      if (ctx) {
        console.log('Canvas 2D context created successfully');
      } else {
        console.error('Failed to get 2D context from canvas');
      }
      resizeCanvas();
      window.addEventListener('resize', resizeCanvas);
    
    // 鼠标事件
    canvas.value.addEventListener('mousedown', (e) => {
      e.preventDefault();
      mousedown = true;
      mx = e.pageX - canvas.value!.offsetLeft;
      my = e.pageY - canvas.value!.offsetTop;
    });
    
    canvas.value.addEventListener('mouseup', (e) => {
      e.preventDefault();
      mousedown = false;
    });
    
    canvas.value.addEventListener('mousemove', (e) => {
      if (mousedown) {
        mx = e.pageX - canvas.value!.offsetLeft;
        my = e.pageY - canvas.value!.offsetTop;
      }
    });
    
    // 触摸事件
    canvas.value.addEventListener('touchstart', (e) => {
      e.preventDefault();
      mousedown = true;
      mx = e.touches[0].pageX - canvas.value!.offsetLeft;
      my = e.touches[0].pageY - canvas.value!.offsetTop;
    });
    
    canvas.value.addEventListener('touchend', (e) => {
      e.preventDefault();
      mousedown = false;
    });
    
    canvas.value.addEventListener('touchmove', (e) => {
      if (mousedown) {
        mx = e.touches[0].pageX - canvas.value!.offsetLeft;
        my = e.touches[0].pageY - canvas.value!.offsetTop;
      }
    });
    
    // 由于使用v-if，组件挂载时show一定为true，直接启动动画
    console.log('Starting fireworks animation on mount (v-if mode)');
    // 确保在下一帧启动动画
    requestAnimationFrame(() => {
      loop();
    });
  }
  }, 100); // 给予足够时间让 DOM 完全渲染
});

onUnmounted(() => {
  console.log('Fireworks component unmounted, cleaning up resources');
  
  // 清理倒计时定时器
  if (timerInterval.value) {
    clearInterval(timerInterval.value);
    timerInterval.value = null;
  }
  
  // 清理动画帧
  if (animationFrameId) {
    console.log('Cancelling animation frame on unmount:', animationFrameId);
    cancelAnimationFrame(animationFrameId);
    animationFrameId = null;
  }
  
  // 移除事件监听器
  window.removeEventListener('resize', resizeCanvas);
  
  // 清空画布和数据
  if (ctx && canvas.value) {
    console.log('Clearing canvas on unmount');
    ctx.clearRect(0, 0, canvas.value.width, canvas.value.height);
    fireworks = [];
    particles = [];
  }
});
</script>

<template>
  <div class="fireworks-container">
    <canvas 
      ref="canvas" 
      class="fireworks-canvas" 
    ></canvas>
    <div v-if="showText" class="fireworks-text">
      <span v-for="(line, idx) in fireworksLines" :key="idx" class="fireworks-line">{{ line }}</span>
    </div>
    <div class="countdown-container">
      <div class="countdown">{{ remainingTime }}s</div>
      <button class="close-button" @click="handleClose">CLOSE</button>
    </div>
  </div>
</template>

<style scoped>
.fireworks-container {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 99999; /* 增加 z-index 确保显示在最上层 */
  background-color: rgba(0, 0, 0, 0.9);
  display: flex;
  justify-content: center;
  align-items: center;
  pointer-events: none; /* 允许点击穿透 */
}

.fireworks-canvas {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: auto; /* 确保画布可以接收鼠标事件 */
  z-index: 100000; /* 确保画布在文本之下但在其他元素之上 */
}

.fireworks-text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  z-index: 100001;
  text-align: center;
  max-width: 80%;
  /* padding: 20px 40px; */
  border-radius: 15px;
  pointer-events: none;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.fireworks-line {
  display: block;
  width: auto;
  font-size: 4rem;
  font-weight: bold;
  color: transparent;
  background-image: linear-gradient(97deg,#0096FF,#BB64FF 42%,#F2416B 74%,#EB7500);
  background-clip: text;
  -webkit-background-clip: text;
  -webkit-mask-clip: text;
  mask-clip: text;
  line-height: 1.2;
  margin: 0.2em 0;
  /* 横向舞台灯光高光动画 */
  -webkit-mask-image: linear-gradient(90deg, transparent 0%, rgba(255,255,255,0.85) 40%, rgba(255,255,255,0.85) 70%, transparent 100%);
  -webkit-mask-size: 200% 100%;
  -webkit-mask-position: 0 0;
  -webkit-animation: shine-mask 2.5s linear infinite;
  mask-image: linear-gradient(90deg, transparent 0%, rgba(255,255,255,0.85) 40%, rgba(255,255,255,0.85) 70%, transparent 100%);
  mask-size: 200% 100%;
  mask-position: 0 0;
  animation: shine-mask 2.5s linear infinite;
}
@-webkit-keyframes shine-mask {
  0% { -webkit-mask-position: 0 0; mask-position: 0 0; }
  100% { -webkit-mask-position: 100% 0; mask-position: 100% 0; }
}
@keyframes shine-mask {
  0% { -webkit-mask-position: 0 0; mask-position: 0 0; }
  100% { -webkit-mask-position: 100% 0; mask-position: 100% 0; }
}

.countdown-container {
  position: absolute;
  top: 20px;
  right: 20px;
  display: flex;
  align-items: center;
  z-index: 100002; /* 确保显示在最上层 */
  pointer-events: auto; /* 确保可以点击 */
}

.countdown {
  background-color: rgba(0, 0, 0, 0.7);
  color: white;
  font-size: 1rem;
  font-weight: bold;
  padding: 8px 12px;
  border-radius: 20px;
  margin-right: 10px;
  min-width: 40px;
  text-align: center;
}

.close-button {
  background-color: rgba(255, 255, 255, 0.2);
  color: white;
  border: 2px solid white;
  border-radius: 20px;
  padding: 8px 15px;
  font-size: 0.9rem;
  font-weight: bold;
  cursor: pointer;
  transition: all 0.3s ease;
  outline: none;
}

.close-button:hover {
  background-color: rgba(255, 255, 255, 0.4);
  transform: scale(1.05);
}

.close-button:active {
  transform: scale(0.95);
}
</style>