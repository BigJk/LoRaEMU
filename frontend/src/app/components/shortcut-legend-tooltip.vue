<template>
	<div class="b mb2 bb pb2 b--white-10">Hotkeys</div>
	<div class="mb2 flex justify-between items-center">
		Change Node Position
		<n-tag size="small" class="mr2"> SHIFT + Drag </n-tag>
	</div>
	<div class="flex justify-between items-center">
		Context Menu
		<n-tag size="small" class="mr2"> Right Click </n-tag>
	</div>
	<div class="b mb2 mt3 bb pb2 b--white-10">Legend</div>
	<div class="mb2 flex justify-between items-center">
		Sending
		<n-tag size="small" :color="{ color: 'lightgreen', textColor: 'black' }" :bordered="false" class="mr2"> Green Line & Node </n-tag>
	</div>
	<div class="mb2 flex justify-between items-center">
		Receiving
		<n-tag size="small" :color="{ color: 'yellow', textColor: 'black' }" :bordered="false" class="mr2"> Yellow Node </n-tag>
	</div>
	<div class="mb2 flex justify-between items-center">
		Collision
		<n-tag size="small" :color="{ color: 'red', textColor: 'white' }" :bordered="false" class="mr2"> Red Line & Node </n-tag>
	</div>
	<div class="mb2 flex justify-between items-center">
		Selected
		<n-tag size="small" :color="{ color: 'cornflowerblue', textColor: 'white' }" :bordered="false" class="mr2"> Blue Line & Node </n-tag>
	</div>
	<div class="mb2 flex justify-between items-center">
		Can Reach
		<n-tag size="small" :bordered="false" class="mr2"> Arrow </n-tag>
	</div>
</template>

<script>
export default {
	name: 'shortcut-legend-tooltip',
};
</script>

<style scoped></style>
