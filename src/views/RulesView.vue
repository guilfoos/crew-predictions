<template>
  <div class="page">
    <h1 class="page-title">
      Rules
    </h1>
    <p class="rules-intro">
      Three ways to be wrong. Pick your poison — actually, you don't have a choice, you're in all of the contests.
    </p>

    <section class="rules-section">
      <h2 class="section-title">
        Aces Radio
      </h2>
      <p class="rules-desc">
        High risk, high reward. Nail the exact scoreline and you're the smartest person in the room. Get it backwards and you owe everyone a pint.
      </p>
      <div class="rules-table">
        <div class="rules-row rules-row--positive">
          <span class="rules-points">+15</span>
          <span class="rules-condition">Exact scoreline — you called it perfectly</span>
        </div>
        <div class="rules-row rules-row--positive">
          <span class="rules-points">+10</span>
          <span class="rules-condition">Correct winner or draw, wrong score</span>
        </div>
        <div class="rules-row rules-row--negative">
          <span class="rules-points">-15</span>
          <span class="rules-condition">Flipped — you predicted the exact same score but reversed (e.g. you said 2-1 Crew win, actual was 1-2 loss)</span>
        </div>
        <div class="rules-row rules-row--zero">
          <span class="rules-points">0</span>
          <span class="rules-condition">Everything else. Wrong result, wrong score, wrong life choices.</span>
        </div>
      </div>
    </section>

    <section class="rules-section">
      <h2 class="section-title">
        Upper 90 Club
      </h2>
      <p class="rules-desc">
        Gentler scoring for the optimists. Points stack — nail all three and you're golden.
      </p>
      <div class="rules-table">
        <div class="rules-row rules-row--positive">
          <span class="rules-points">+1</span>
          <span class="rules-condition">Correct result — win, draw, or loss</span>
        </div>
        <div class="rules-row rules-row--positive">
          <span class="rules-points">+1</span>
          <span class="rules-condition">Exact Columbus Crew goal tally — you predicted the right number of Crew goals</span>
        </div>
        <div class="rules-row rules-row--positive">
          <span class="rules-points">+1</span>
          <span class="rules-condition">Exact opponent goal tally — you predicted the right number of goals for the other lot</span>
        </div>
      </div>
      <p class="rules-note">
        Points stack. Nail all three and you're at +3.
      </p>
    </section>

    <section class="rules-section">
      <h2 class="section-title">
        Grouchy™
      </h2>
      <p class="rules-desc">
        The prediction game from the most awful forum in the most awful place to talk soccer since 2000.
      </p>
      <div class="rules-table">
        <div class="rules-row rules-row--positive">
          <span class="rules-points">+1</span>
          <span class="rules-condition">Win by 2+ — you predicted a comfortable Crew victory and got one</span>
        </div>
        <div class="rules-row rules-row--positive">
          <span class="rules-points">+1</span>
          <span class="rules-condition">Win by 1 — a nervy Crew win and you called it</span>
        </div>
        <div class="rules-row rules-row--positive">
          <span class="rules-points">+1</span>
          <span class="rules-condition">Draw — 90 minutes of nothing and you predicted it</span>
        </div>
        <div class="rules-row rules-row--positive">
          <span class="rules-points">+1</span>
          <span class="rules-condition">Lose by 1 — a heartbreaker and you saw it coming</span>
        </div>
        <div class="rules-row rules-row--positive">
          <span class="rules-points">+1</span>
          <span class="rules-condition">Lose by 2+ — a disaster and somehow you predicted that too</span>
        </div>
        <div class="rules-row rules-row--zero">
          <span class="rules-points">0</span>
          <span class="rules-condition">Wrong category. Touch grass.</span>
        </div>
      </div>
      <p class="rules-note">
        Categories are based on Columbus Crew margin. Predict the right bucket and you get the point — exact score doesn't matter.
      </p>
    </section>

    <section class="memorial-section">
      <div class="memorial-icon-large">⚫</div>
      <h2 class="memorial-title">In Loving Memory</h2>
      <p class="memorial-name">Mike Cornell</p>
      <p class="memorial-handle">mcornell.dev</p>
      <p class="memorial-message">
        This site was built by Mike as a labor of love for the Crew and its fans.
        He passed away in November 2026, but his work lives on.
        Every prediction, every leaderboard update, and every match result honors his vision
        and his passion for the Black & Gold.
      </p>
      <p class="memorial-sub">
        Pick your scores. Be wrong in public. It's tradition — just like Mike would have wanted.
      </p>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'

onMounted(() => { document.title = 'Rules — Crew Predictions' })
</script>

<style scoped>
.memorial-section {
  margin-top: 4rem;
  padding: 2.5rem 2rem;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  border-top: 3px solid #fedd00;
  text-align: center;
}

.memorial-icon-large {
  font-size: 3rem;
  margin-bottom: 1rem;
  opacity: 0.8;
}

.memorial-title {
  font-family: 'Barlow Condensed', sans-serif;
  font-weight: 800;
  font-size: 1.8rem;
  color: #aaa;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  margin-bottom: 0.5rem;
}

.memorial-name {
  font-family: 'Barlow Condensed', sans-serif;
  font-weight: 800;
  font-size: 2.5rem;
  color: #fedd00;
  letter-spacing: 0.04em;
  margin-bottom: 0.25rem;
}

.memorial-handle {
  font-family: 'DM Mono', monospace;
  font-size: 0.9rem;
  color: #fedd00;
  margin-bottom: 1.5rem;
  opacity: 0.8;
}

.memorial-message {
  font-size: 1rem;
  color: #aaa;
  line-height: 1.7;
  max-width: 540px;
  margin: 0 auto 1rem;
}

.memorial-sub {
  font-family: 'DM Mono', monospace;
  font-size: 0.85rem;
  color: #888;
  font-style: italic;
}

@media (max-width: 600px) {
  .memorial-section {
    padding: 1.5rem 1rem;
    margin-top: 2.5rem;
  }

  .memorial-icon-large {
    font-size: 2.2rem;
  }

  .memorial-title {
    font-size: 1.4rem;
  }

  .memorial-name {
    font-size: 1.8rem;
  }

  .memorial-handle {
    font-size: 0.8rem;
  }

  .memorial-message {
    font-size: 0.9rem;
    line-height: 1.6;
  }

  .memorial-sub {
    font-size: 0.75rem;
  }
}
</style>
