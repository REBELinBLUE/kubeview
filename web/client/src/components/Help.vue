<template>
  <b-modal id="aboutModal" title="Help with KubeView" header-bg-variant="info" header-text-variant="dark" ok-only>
    <div class="text-center">
      <img src="../assets/logo.png" width="100">

      <p>KubeView displays the current state of your cluster. Each resource is represented by a different icon, click them to see information on the resource.</p>
      <p>Solid lines indicate that one resource <i>references</i> another in the direction of the arrow, a dashed line indicates that one resources <i>creates</i> another in the direction of the arrow.</p>

      <b-button href="https://github.com/REBELinBLLUE/kubeview" target="_blank" variant="success">GitHub Project</b-button>

      <p id="credit">Based on <a target="_blank" href="https://github.com/benc-uk/kubeview">KubeView by Ben Coleman</a></p>
    </div>
  </b-modal>
</template>

<style>
  #credit {
    margin: 15px auto 0px auto;
  }
</style>